package buildings

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/email_h"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/isr"
	"github.com/yaz/kyo-repo/internal/receiptPdf"
	"github.com/yaz/kyo-repo/internal/reserveFunds"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
)

const _PATH = "/api/buildings"
const _SEARCH = _PATH + "/search"
const _UPLOAD_BACKUP_FORM = _PATH + "/uploadBackupForm"
const _UPLOAD_BACKUP = _PATH + "/upload/backup"

func Routes(holder *api.RouterHolder) {

	holder.GET(_SEARCH, search, api.BUILDINGS_READ)
	holder.DELETE(_PATH+"/{id}", buildingDelete, api.BuildingsDeleteRecaptchaAction, api.BUILDINGS_WRITE)
	holder.PUT(_PATH, buildingPut, api.BuildingsUpsertRecaptchaAction, api.BUILDINGS_WRITE)
	holder.GET(_PATH+"/formData", formData, api.BUILDINGS_WRITE)
	holder.GET(_UPLOAD_BACKUP_FORM, getUploadBackupForm, api.BUILDINGS_READ)
	holder.POST(_UPLOAD_BACKUP, uploadBackup, api.BuildingsUploadBackupRecaptchaAction, api.BUILDINGS_UPLOAD_BACKUP)
}

func search(w http.ResponseWriter, r *http.Request) {

	requestQuery := RequestQuery{
		LastCreatedAt: util.GetQueryParamAsTimestamp(r, "next_page"),
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
	id := mux.Vars(r)["id"]
	if id == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var dest string
	err := util.Decode(id, &dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	counters, err := deleteAndReturnCounters(dest)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer receiptPdf.PublishBuilding(r.Context(), dest)
	defer isr.Invoke(r.Context())

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

		validate, err := util.GetValidator()
		if err != nil {
			log.Printf("Error getting validator: %v", err)
			response.errorStr = err.Error()
			return response
		}

		isUpdate := request.Key != nil
		if isUpdate {
			err := util.Decode(*request.Key, &request.Id)
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

		currenciesToShowAmountToPayMap := make(map[string]bool)
		for _, currency := range request.CurrenciesToShowAmountToPay {
			currenciesToShowAmountToPayMap[currency] = true
		}

		if len(currenciesToShowAmountToPayMap) == 0 {
			currenciesToShowAmountToPayMap[request.MainCurrency] = true
		}

		currenciesToShowAmountToPay := strings.Join(slices.Collect(maps.Keys(currenciesToShowAmountToPayMap)), ",")

		debtsCurrenciesToShowMap := make(map[string]bool)
		for _, currency := range request.DebtsCurrenciesToShow {
			debtsCurrenciesToShowMap[currency] = true
		}

		if len(debtsCurrenciesToShowMap) == 0 {
			debtsCurrenciesToShowMap[request.DebtCurrency] = true
		}

		debtsCurrenciesToShow := strings.Join(slices.Collect(maps.Keys(debtsCurrenciesToShowMap)), ",")

		building := model.Buildings{
			ID:                          request.Id,
			Name:                        request.Name,
			Rif:                         request.Rif,
			MainCurrency:                request.MainCurrency,
			DebtCurrency:                request.DebtCurrency,
			CurrenciesToShowAmountToPay: currenciesToShowAmountToPay,
			DebtsCurrenciesToShow:       debtsCurrenciesToShow,
			FixedPay:                    request.FixedPay,
			FixedPayAmount:              request.FixedPayAmount,
			RoundUpPayments:             request.RoundUpPayments,
			EmailConfig:                 request.EmailConfig,
		}

		if isUpdate {
			err = update(building)
			if err == nil {
				defer receiptPdf.PublishBuilding(r.Context(), building.ID)
			}

		} else {
			err = insert(building)
			defer isr.Invoke(r.Context())
		}

		if err != nil {
			response.errorStr = err.Error()
			return response
		}

		newKey := util.Encode(building.ID)
		response.key = newKey
		tmp := !isUpdate
		response.createdNew = &tmp
		return response
	}

	response := upsert()

	if response.createdNew != nil && *response.createdNew && response.key != nil {
		redirectUrl := "/buildings/edit/" + *response.key

		err := api.AnchorClickInitView(redirectUrl).Render(r.Context(), w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//w.Header().Add("HX-Location", redirectUrl)
		//w.WriteHeader(http.StatusOK)
		return
	}

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func formData(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	emailMap, err := email_h.GetConfigs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	emailConfigs := make([]EmailConfig, 0)
	for key := range emailMap {
		emailConfigs = append(emailConfigs, EmailConfig{
			key:   key,
			email: emailMap[key].Username,
		})
	}

	slices.SortFunc(emailConfigs, func(a, b EmailConfig) int {
		return strings.Compare(a.email, b.email)
	})

	updateParams := UpdateParams{
		Apts: make([]apartments.Apt, 0),
	}

	formDto := FormDto{
		emailConfigs: emailConfigs,
	}

	if idParam != "" {
		var id string
		err := util.Decode(idParam, &id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var wg sync.WaitGroup
		wg.Add(4)
		errChan := make(chan error, 4)

		go func() {
			defer wg.Done()
			building, err := selectById(id)
			if err != nil {
				errChan <- err
				return
			}
			if building == nil {
				errChan <- errors.New("building not found")
				return
			}

			formDto.Key = &idParam
			updateParams.ID = building.ID
			updateParams.Name = building.Name
			updateParams.Rif = building.Rif
			updateParams.MainCurrency = building.MainCurrency
			updateParams.DebtCurrency = building.DebtCurrency
			updateParams.CurrenciesToShowAmountToPay = strings.Split(building.CurrenciesToShowAmountToPay, ",")
			updateParams.DebtsCurrenciesToShow = strings.Split(building.DebtsCurrenciesToShow, ",")
			updateParams.FixedPay = building.FixedPay
			updateParams.FixedPayAmount = building.FixedPayAmount
			updateParams.RoundUpPayments = building.RoundUpPayments
			updateParams.EmailConfig = building.EmailConfig

		}()

		go func() {
			defer wg.Done()
			reserveFundFormDto, err := reserveFunds.GetFormDto(id, "")
			if err != nil {
				errChan <- err
				return
			}

			formDto.reserveFundFormDto = *reserveFundFormDto
		}()

		go func() {
			defer wg.Done()
			extraChargesFormDto, err := extraCharges.GetBuildingFormDto(id)
			if err != nil {
				errChan <- err
				return
			}

			formDto.extraChargesFormDto = *extraChargesFormDto
		}()

		go func() {
			defer wg.Done()
			apts, err := apartments.SelectNumberAndNameByBuildingId(id)
			if err != nil {
				errChan <- err
				return
			}

			updateParams.Apts = apts
		}()

		wg.Wait()
		close(errChan)

		err = util.HasErrors(errChan)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		byteArray, err := json.Marshal(updateParams)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		base64Str := base64.URLEncoding.EncodeToString(byteArray)
		formDto.UpdateParams = &base64Str

	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/html")
	err = FormView(formDto).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getUploadBackupForm(w http.ResponseWriter, r *http.Request) {

	component, err := api.BuildUploadForm(r, "BACKUPS/BUILDINGS/")

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
	Building     buildingDto                   `json:"building"`
	ReserveFunds []reserveFundDto              `json:"reserve_funds"`
	ExtraCharges []extraCharges.ExtraChargeDto `json:"extra_charges"`
}

type buildingDto struct {
	Id                          string   `json:"id"`
	Name                        string   `json:"name"`
	Rif                         string   `json:"rif"`
	MainCurrency                string   `json:"main_currency"`
	DebtCurrency                string   `json:"debt_currency"`
	CurrenciesToShowAmountToPay []string `json:"currencies_to_show_amount_to_pay"`
	FixedPay                    bool     `json:"fixed_pay"`
	FixedPayAmount              float64  `json:"fixed_pay_amount"`
	RoundUpPayments             bool     `json:"round_up_payments"`
}

type reserveFundDto struct {
	BuildingID    string  `json:"building_id"`
	Name          string  `json:"name"`
	Fund          float64 `json:"fund"`
	Expense       float64 `json:"expense"`
	Pay           float64 `json:"pay"`
	Active        bool    `json:"active"`
	Type          string  `json:"type"`
	ExpenseType   string  `json:"expense_type"`
	AddToExpenses bool    `json:"add_to_expenses"`
}

func uploadBackup(w http.ResponseWriter, r *http.Request) {

	component, err := api.ProcessUploadBackup(r, "/buildings", ProcessDecoder)

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

func ProcessDecoder(decoder *json.Decoder) (int64, error) {
	var dto []BuildingRecord
	err := decoder.Decode(&dto)
	if err != nil {
		log.Printf("Error decoding json: %s", err)
		return 0, err
	}

	buildings := make([]model.Buildings, len(dto))
	var reserveFundArray []model.ReserveFunds
	var extraChargeArray []model.ExtraCharges

	configs, err := email_h.GetConfigs()
	if err != nil {
		return 0, err
	}

	getFirst := func() string {
		for key := range configs {
			return key
		}
		return ""
	}

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
			EmailConfig:                 getFirst(),
		}

		for _, reserveFund := range record.ReserveFunds {
			reserveFundArray = append(reserveFundArray, model.ReserveFunds{
				BuildingID:    reserveFund.BuildingID,
				Name:          reserveFund.Name,
				Fund:          reserveFund.Fund,
				Expense:       reserveFund.Expense,
				Pay:           reserveFund.Pay,
				Active:        reserveFund.Active,
				Type:          reserveFund.Type,
				ExpenseType:   reserveFund.ExpenseType,
				AddToExpenses: reserveFund.AddToExpenses,
			})
		}

		for _, extraCharge := range record.ExtraCharges {
			var builder strings.Builder
			for idx, apt := range extraCharge.Apartments {
				builder.WriteString(apt.Number)
				if idx < len(extraCharge.Apartments)-1 {
					builder.WriteString(",")
				}
			}

			extraChargeArray = append(extraChargeArray, model.ExtraCharges{
				BuildingID:      extraCharge.BuildingID,
				ParentReference: extraCharge.ParentReference,
				Type:            extraCharge.Type,
				Description:     extraCharge.Description,
				Amount:          extraCharge.Amount,
				Currency:        extraCharge.Currency,
				Active:          extraCharge.Active,
				Apartments:      builder.String(),
			})
		}
	}

	rowsAffected, err := insertBackup(buildings)
	if err != nil {
		return 0, err
	}

	_, err = reserveFunds.InsertBackup(reserveFundArray)
	if err != nil {
		return 0, err
	}

	_, err = extraCharges.InsertBulk(extraChargeArray)
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
