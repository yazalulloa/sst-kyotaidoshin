package debts

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/receiptPdf"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"net/http"
)

const _PATH = "/api/debts"

func Routes(holder *api.RouterHolder) {

	holder.PUT(_PATH, debtPut, api.DebtsUpsertRecaptchaAction, api.RECEIPTS_WRITE)
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
		err = util.Decode(request.Key, &keys)
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

		years := make([]YearWithMonths, len(request.DebtMonths))

		for i, debtMonth := range request.DebtMonths {
			var year YearWithMonths
			err := json.Unmarshal([]byte(debtMonth), &year)
			if err != nil {
				log.Printf("Error decoding debt month: %s %v", debtMonth, err)
				response.errorStr = err.Error()
				return response
			}
			years[i] = year
		}

		monthlyDebt := MonthlyDebt{
			Amount: request.DebtMonthsTotal,
			Years:  years,
		}

		months, err := json.Marshal(monthlyDebt)
		if err != nil {
			log.Printf("Error encoding monthly debt: %v", err)
			response.errorStr = err.Error()
			return response
		}

		debt := model.Debts{
			BuildingID:                    keys.BuildingID,
			ReceiptID:                     keys.ReceiptID,
			AptNumber:                     keys.AptNumber,
			Receipts:                      request.Receipts,
			Amount:                        request.Amount,
			Months:                        string(months),
			PreviousPaymentAmount:         request.PreviousPaymentAmount,
			PreviousPaymentAmountCurrency: request.PreviousPaymentAmountCurrency,
		}

		repository := NewRepository(r.Context())

		_, err = repository.update(debt)

		if err != nil {
			log.Printf("Error inserting/updating debt: %v", err)
			response.errorStr = err.Error()
			return response
		}

		defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingID, keys.ReceiptID)

		item, err := toItem(&debt, &keys.CardId)
		tmp := true
		item.isUpdate = &tmp
		if err != nil {
			log.Printf("Error getting item: %v", err)
			response.errorStr = err.Error()
			return response
		}

		formDto, err := repository.GetFormDto(keys.BuildingID, keys.ReceiptID)
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
