package buildings

import (
	"db/gen/model"
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

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", buildingDelete).Methods("DELETE")
	server.HandleFunc(_PATH, buildingPut).Methods("PUT")
	server.HandleFunc(_PATH+"/formData", formData).Methods("GET")
}

func search(w http.ResponseWriter, r *http.Request) {

	requestQuery := RequestQuery{
		LastCreatedAt: nil,
		Limit:         30,
		SortOrder:     util.SortOrderTypeDESC,
	}

	response, err := getTableResponse(requestQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = Search(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func buildingDelete(w http.ResponseWriter, r *http.Request) {

	id := api.GetQueryParamAsString(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err := deleteById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func buildingPut(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	decoder := form.NewDecoder()
	var request FormRequest
	err = decoder.Decode(&request, r.Form)

	if err != nil {
		log.Printf("Error decoding form: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	validate := validator.New(validator.WithRequiredStructEnabled())

	isUpdate := request.Key != nil
	log.Printf("Is update: %v", isUpdate)
	if isUpdate {
		err := api.Decode(*request.Key, &request.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = validate.Struct(request)
	if err != nil {
		// Validation failed, handle the error
		errors := err.(validator.ValidationErrors)
		for _, valErr := range errors {
			log.Printf("Validation error: %v", valErr)
		}
		http.Error(w, fmt.Sprintf("Validation error: %s", errors), http.StatusBadRequest)
		return
	}

	if !isUpdate {
		exists, err := idExists(request.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, "Id already exists", http.StatusBadRequest)
			return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(w, "NotImplemented", http.StatusNotImplemented)
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
	Key                         *string  `form:"Key"`
	Id                          string   `form:"id" validate:"required_if=Key nil,min=3,max=20,alphanumunicode"`
	Name                        string   `form:"name" validate:"required,min=3,max=100"`
	Rif                         string   `form:"rif" validate:"required,min=7,max=20"`
	MainCurrency                string   `form:"mainCurrency" validate:"required,oneof=USD VED"`
	DebtCurrency                string   `form:"debtCurrency" validate:"required,oneof=USD VED"`
	CurrenciesToShowAmountToPay []string `form:"currenciesToShowAmountToPay" validate:"dive,oneof=USD VED"`
	RoundUpPayments             bool     `form:"roundUpPayments"`
	FixedPay                    bool     `form:"fixedPay"`
	FixedPayAmount              float64  `form:"fixedPayAmount" validate:"required_if=fixedPay true,gt=0"`
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
