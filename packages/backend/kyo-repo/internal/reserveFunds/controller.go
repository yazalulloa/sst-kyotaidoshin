package reserveFunds

import (
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/receiptPdf"
	"github.com/yaz/kyo-repo/internal/util"
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

	service := NewService(r.Context())

	var idToLookup int32
	var cardIdStr *string
	if isUpdate {
		_, err = service.repo.update(reserveFund)
		idToLookup = *keys.Id
		cardIdStr = &keys.CardId
	} else {
		lastInsertId, err := service.repo.insert(reserveFund)
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

	item, err := service.getItem(idToLookup, keys.ReceiptId, cardIdStr)
	if err != nil {
		log.Printf("Error getting Item: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	count, err := service.repo.CountByBuilding(keys.BuildingId)
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

	_, err = NewRepository(r.Context()).deleteById(*keys.Id)
	if err != nil {
		return "", keys, err
	}

	defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingId)

	return key, keys, nil
}
