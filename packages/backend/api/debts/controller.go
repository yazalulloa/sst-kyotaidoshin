package debts

import (
	"db/gen/model"
	"fmt"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/debts"

func Routes(server *mux.Router) {

	server.HandleFunc(_PATH, debtPut).Methods("PUT")
}

func debtPut(w http.ResponseWriter, r *http.Request) {

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

		var keys Keys
		err = api.Decode(request.Key, &keys)
		if err != nil {
			log.Printf("Error decoding key: %v", err)
			response.errorStr = err.Error()
			return response
		}

		validate, err := util.GetValidator()
		if err != nil {
			log.Printf("Error getting validator: %v", err)
			response.errorStr = err.Error()
			return response
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
		var monthBuilder strings.Builder
		for i, v := range request.Months {
			monthBuilder.WriteString(fmt.Sprint(v))
			if i < len(request.Months)-1 {
				monthBuilder.WriteString(",")
			}
		}

		debt := model.Debts{
			BuildingID:                    keys.BuildingID,
			ReceiptID:                     keys.ReceiptID,
			AptNumber:                     keys.AptNumber,
			Receipts:                      request.Receipts,
			Amount:                        request.Amount,
			Months:                        monthBuilder.String(),
			PreviousPaymentAmount:         request.PreviousPaymentAmount,
			PreviousPaymentAmountCurrency: request.PreviousPaymentAmountCurrency,
		}

		_, err = update(debt)

		if err != nil {
			log.Printf("Error inserting/updating debt: %v", err)
			response.errorStr = err.Error()
			return response
		}

		item, err := toItem(&debt, &keys.CardId)
		tmp := true
		item.isUpdate = &tmp
		if err != nil {
			log.Printf("Error getting item: %v", err)
			response.errorStr = err.Error()
			return response
		}

		formDto, err := GetFormDto(keys.BuildingID, keys.ReceiptID)
		if err != nil {
			log.Printf("Error getting totals: %v", err)
			response.errorStr = err.Error()
			return response
		}

		response.item = item
		response.Totals = &formDto.Totals

		return response
	}

	response := upsert()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
