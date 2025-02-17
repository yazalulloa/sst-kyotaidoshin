package buildings

import (
	"db/gen/model"
	"encoding/json"
	"fmt"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"kyotaidoshin/util"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
)

const _PATH = "/api/buildings"
const _SEARCH = _PATH + "/search"
const _UPLOAD_BACKUP_FORM = _PATH + "/uploadBackupForm"
const _UPLOAD_BACKUP = _PATH + "/upload/backup"

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", buildingDelete).Methods("DELETE")
	server.HandleFunc(_PATH, buildingPut).Methods("PUT")
	server.HandleFunc(_PATH+"/formData", formData).Methods("GET")
	server.HandleFunc(_UPLOAD_BACKUP_FORM, getUploadBackupForm).Methods("GET")
	server.HandleFunc(_UPLOAD_BACKUP, uploadBackup).Methods("GET")
}

func search(w http.ResponseWriter, r *http.Request) {

	requestQuery := RequestQuery{
		LastCreatedAt: api.GetQueryParamAsTimestamp(r, "next_page"),
		Limit:         31,
		SortOrder:     util.SortOrderTypeDESC,
	}

	response, err := getTableResponse(requestQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := response.Results
	var nextPageUrl string
	if len(results) == requestQuery.Limit {
		results = results[:len(results)-1]
		last := results[len(results)-1]
		var sb strings.Builder
		time := *last.Item.CreatedAt
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%d", time.UnixMilli()))

		nextPageUrl = sb.String()
	}

	response.NextPageUrl = nextPageUrl
	response.Results = results

	err = Search(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func buildingDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var dest string
	err := api.Decode(id, &dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	counters, err := deleteAndReturnCounters(dest)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = CountersView(*counters).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func buildingPut(w http.ResponseWriter, r *http.Request) {

	upsert := func() FormResponse {

		response := FormResponse{}

		err := r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form: %v", err)
			response.errorStr = err.Error()
			return response
		}

		decoder := form.NewDecoder()
		var request FormRequest
		err = decoder.Decode(&request, r.Form)

		if err != nil {
			log.Printf("Error decoding form: %v", err)
			response.errorStr = err.Error()
			return response
		}

		validate := validator.New(validator.WithRequiredStructEnabled())

		isUpdate := request.Key != nil
		log.Printf("Is update: %v", isUpdate)
		if isUpdate {
			err := api.Decode(*request.Key, &request.Id)
			if err != nil {
				response.errorStr = fmt.Sprintf("Error decoding key: %v", err)
				return response
			}
		}

		err = validate.Struct(request)
		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			for _, valErr := range errors {
				log.Printf("Validation error: %v", valErr)
			}
			response.errorStr = fmt.Sprintf("Validation error: %s", errors)
			return response
		}

		if request.FixedPay && request.FixedPayAmount <= 0 {
			response.errorStr = "Fixed pay amount must be greater than 0"
			return response
		}

		if !isUpdate {
			exists, err := idExists(request.Id)
			if err != nil {
				response.errorStr = err.Error()
				return response
			}

			if exists {
				response.errorStr = fmt.Sprintf("ID %s already exists", request.Id)
				return response
			}
		}

		currencies := make(map[string]bool)
		for _, currency := range request.CurrenciesToShowAmountToPay {
			currencies[currency] = true
		}

		if len(currencies) == 0 {
			currencies[request.MainCurrency] = true
		}

		currenciesToShowAmountToPay := strings.Join(slices.Collect(maps.Keys(currencies)), ",")

		var fixedPayAmount *float64
		if request.FixedPay {
			fixedPayAmount = &request.FixedPayAmount
		} else {
			fixedPayAmount = nil
		}

		building := model.Buildings{
			ID:                          request.Id,
			Name:                        request.Name,
			Rif:                         request.Rif,
			MainCurrency:                request.MainCurrency,
			DebtCurrency:                request.DebtCurrency,
			CurrenciesToShowAmountToPay: currenciesToShowAmountToPay,
			FixedPay:                    request.FixedPay,
			FixedPayAmount:              fixedPayAmount,
			RoundUpPayments:             request.RoundUpPayments,
			EmailConfig:                 request.EmailConfig,
		}

		if isUpdate {
			err = update(building)
		} else {
			err = insert(building)
		}

		if err != nil {
			response.errorStr = err.Error()
			return response
		}

		newKey := api.Encode(building.ID)
		response.key = newKey
		tmp := !isUpdate
		response.createdNew = &tmp
		log.Printf("Created new: %v", *response.createdNew)
		return response
	}

	response := upsert()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

const idMinLen = 3
const idMaxLen = 20
const nameMinLen = 3
const nameMaxLen = 100
const rifMinLen = 7
const rifMaxLen = 20
const currencyMaxLen = 3
const fixedPayAmountMaxLen = 18

type FormRequest struct {
	Key                         *string  `form:"key"`
	Id                          string   `form:"id" validate:"required_if=Key nil,min=3,max=20,alphanumunicode"`
	Name                        string   `form:"name" validate:"required,min=3,max=100"`
	Rif                         string   `form:"rif" validate:"required,min=7,max=20"`
	MainCurrency                string   `form:"mainCurrency" validate:"required,oneof=USD VED"`
	DebtCurrency                string   `form:"debtCurrency" validate:"required,oneof=USD VED"`
	CurrenciesToShowAmountToPay []string `form:"currenciesToShowAmountToPay" validate:"dive,oneof=USD VED"`
	RoundUpPayments             bool     `form:"roundUpPayments"`
	FixedPay                    bool     `form:"fixedPay"`
	FixedPayAmount              float64  `form:"fixedPayAmount" validate:"required_if=fixedPay true"`
	EmailConfig                 string   `form:"emailConfig" validate:"required"`
}

func formData(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	formDto := FormDto{
		isEdit: false,
		emailConfigs: []EmailConfig{
			{
				id:    "test",
				key:   "test",
				email: "test@gmail.com",
			},
			{
				id:    "test2",
				key:   "test2",
				email: "test2@gmail.com",
			},
		},
		currencies:                  util.HtmlCurrencies(),
		currenciesToShowAmountToPay: "[]",
	}

	if idParam != "" {
		var id string
		err := api.Decode(idParam, &id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		building, err := selectById(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if building == nil {
			http.Error(w, "Building not found", http.StatusNotFound)
			return
		}

		formDto.isEdit = true
		formDto.building = building
		formDto.key = &idParam
		formDto.currenciesToShowAmountToPay = util.StringArrayToString(strings.Split(building.CurrenciesToShowAmountToPay, ","))

	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/html")
	err := FormView(formDto).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getUploadBackupForm(w http.ResponseWriter, r *http.Request) {

	component, err := api.BuildUploadForm(r.Context(), _UPLOAD_BACKUP[1:], "buildings")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type BuildingRecord struct {
	Building buildingDto `json:"building"`
}

type buildingDto struct {
	Id                          string   `json:"id"`
	Name                        string   `json:"name"`
	Rif                         string   `json:"rif"`
	MainCurrency                string   `json:"main_currency"`
	DebtCurrency                string   `json:"debt_currency"`
	CurrenciesToShowAmountToPay []string `json:"currencies_to_show_amount_to_pay"`
	FixedPay                    bool     `json:"fixed_pay"`
	FixedPayAmount              *float64 `json:"fixed_pay_amount"`
	RoundUpPayments             bool     `json:"round_up_payments"`
}

func uploadBackup(w http.ResponseWriter, r *http.Request) {

	component, err := api.ProcessUploadBackup(r, _UPLOAD_BACKUP_FORM, "buildings-updater", "update-buildings",
		func(decoder *json.Decoder) (int64, error) {
			var dto []BuildingRecord
			err := decoder.Decode(&dto)
			if err != nil {
				log.Printf("Error decoding json: %s", err)
				return 0, err
			}

			buildings := make([]model.Buildings, len(dto))
			for i, record := range dto {
				buildings[i] = model.Buildings{
					ID:                          record.Building.Id,
					Name:                        record.Building.Name,
					Rif:                         record.Building.Rif,
					MainCurrency:                record.Building.MainCurrency,
					DebtCurrency:                record.Building.DebtCurrency,
					CurrenciesToShowAmountToPay: strings.Join(record.Building.CurrenciesToShowAmountToPay, ","),
					FixedPay:                    record.Building.FixedPay,
					FixedPayAmount:              record.Building.FixedPayAmount,
					RoundUpPayments:             record.Building.RoundUpPayments,
					EmailConfig:                 "test",
				}
			}

			return insertBackup(buildings)
		})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
