package reserveFunds

import (
	"fmt"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"kyo-repo/internal/db/gen/model"
	"kyo-repo/internal/receiptPdf"
	"kyo-repo/internal/util"
	"log"
	"net/http"
)

const PATH = "/api/reserveFunds"

func Upsert(r *http.Request) FormResponse {

	response := FormResponse{}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	decoder := form.NewDecoder()
	var request FormRequest
	err = decoder.Decode(&request, r.Form)

	if err != nil {
		log.Printf("Error decoding form: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	var keys Keys
	response.Keys = &keys
	err = util.Decode(request.Key, &keys)
	if err != nil {
		log.Printf("Error decoding key: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	validate, _ := util.GetValidator()
	err = validate.Struct(request)
	if err != nil {
		// Validation failed, handle the error
		errors := err.(validator.ValidationErrors)
		for _, valErr := range errors {
			log.Printf("Validation error: %v", valErr)
		}
		response.ErrorStr = fmt.Sprintf("Validation error: %s", errors)
		return response
	}

	reserveFund := model.ReserveFunds{
		ID:            keys.Id,
		BuildingID:    keys.BuildingId,
		Name:          request.Name,
		Fund:          request.Fund,
		Expense:       request.Expense,
		Pay:           request.Pay,
		Active:        request.Active,
		Type:          request.Type,
		ExpenseType:   request.ExpenseType,
		AddToExpenses: request.AddToExpenses,
	}

	isUpdate := keys.Id != nil

	var idToLookup int32
	var cardIdStr *string
	if isUpdate {
		_, err = update(reserveFund)
		idToLookup = *keys.Id
		cardIdStr = &keys.CardId
	} else {
		lastInsertId, err := insert(reserveFund)
		if err == nil {
			idToLookup = int32(lastInsertId)
		}
	}

	if err != nil {
		log.Printf("Error inserting/updating reserveFund: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingId)

	item, err := getItem(idToLookup, keys.ReceiptId, cardIdStr)
	if err != nil {
		log.Printf("Error getting Item: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	count, err := CountByBuilding(keys.BuildingId)
	if err != nil {
		log.Printf("Error getting count: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	response.Item = item
	response.Item.isUpdate = &isUpdate
	response.counter = count

	return response

}

func DeleteAndReturnKeys(r *http.Request) (string, Keys, error) {
	key := mux.Vars(r)["key"]
	var keys Keys
	err := util.Decode(key, &keys)
	if err != nil {
		return "", keys, err
	}

	if keys.Id == nil {
		return "", keys, fmt.Errorf("id is required")
	}

	_, err = deleteById(*keys.Id)
	if err != nil {
		return "", keys, err
	}

	defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingId)

	return key, keys, nil
}
