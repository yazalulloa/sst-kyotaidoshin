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

	err = validate.Struct(request)
	if err != nil {
		// Validation failed, handle the error
		errors := err.(validator.ValidationErrors)
		http.Error(w, fmt.Sprintf("Validation error: %s", errors), http.StatusBadRequest)
		return
	}

	exists, err := idExists(request.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "Id already exists", http.StatusBadRequest)
		return
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
	}

	err = insert(building)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//err := r.ParseForm()
	//if err != nil {
	//	log.Printf("Error parsing form: %v", err)
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//
	//id := r.Form["id"]
	//name := r.Form["name"]
	//validate := func() error {
	//
	//}
	//
	//if len(id) > idMaxLen {
	//	http.Error(w, "Id too long", http.StatusBadRequest)
	//	return
	//}
	//
	//if len(name) > nameMaxLen {
	//	http.Error(w, "Name too long", http.StatusBadRequest)
	//	return
	//}
	//
	//rif := r.Form["rif"]
	//if len(rif) > rifMaxLen {
	//	http.Error(w, "Rif too long", http.StatusBadRequest)
	//	return
	//}
	//
	//mainCurrency := r.Form["main_currency"]
	//mainCurrency := r.FormValue("main_currency")
	//debtCurrency := r.FormValue("debt_currency")
	//currenciesToShowAmountToPay := r.Form["currenciesToShowAmountToPay"]
	////currenciesToShowAmountToPay := r.FormValue("currenciesToShowAmountToPay")
	//roundUpPayments := r.FormValue("roundUpPayments")
	//fixedPay := r.FormValue("fixedPay")
	//fixedPayAmount := r.FormValue("fixedPayAmount")
	//emailConfig := r.FormValue("emailConfig")
	//
	//log.Printf("Id is %v", id)
	//log.Printf("Name is %v", name)
	//log.Printf("Rif is %v", rif)
	//log.Printf("MainCurrency is %v", mainCurrency)
	//log.Printf("DebtCurrency is %v", debtCurrency)
	//log.Printf("CurrenciesToShowAmountToPay is %v", currenciesToShowAmountToPay)
	//log.Printf("RoundUpPayments is %v", roundUpPayments)
	//log.Printf("FixedPay is %v", fixedPay)
	//log.Printf("FixedPayAmount is %v", fixedPayAmount)
	//log.Printf("EmailConfig is %v", emailConfig)

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
	Id                          string   `form:"id" validate:"required,min=3,max=20,alphanumunicode"`
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
