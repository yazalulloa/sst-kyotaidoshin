package expenses

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

const PATH = "/api/expenses"

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
	err = util.Decode(request.Key, &keys)
	if err != nil {
		log.Printf("Error decoding key: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	validate, err := util.GetValidator()
	if err != nil {
		log.Printf("Error getting validator: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

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

	expense := model.Expenses{
		BuildingID:  keys.BuildingID,
		ReceiptID:   keys.ReceiptID,
		ID:          keys.ID,
		Description: request.Description,
		Amount:      request.Amount,
		Currency:    request.Currency,
		Type:        request.Type,
	}

	isUpdate := keys.ID != nil

	repository := NewRepository(r.Context())

	if isUpdate {
		_, err = repository.update(expense)
	} else {
		lastInsertId, err := repository.insert(expense)
		if err == nil {
			id := int32(lastInsertId)
			keys.ID = &id
		}
	}

	if err != nil {
		log.Printf("Error upserting: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingID, keys.ReceiptID)

	item, err := toItem(&expense, keys.CardId)
	if err != nil {
		log.Printf("Error converting to Item: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	var counter *int64

	if !isUpdate {
		count, err := repository.countByReceipt(keys.ReceiptID)
		if err != nil {
			log.Printf("Error getting count: %v", err)
			response.ErrorStr = err.Error()
			return response
		}
		counter = &count
	}

	response.Item = item
	response.Item.isUpdate = &isUpdate
	response.counter = counter

	return response
}

func DeleteAndReturnKeys(r *http.Request) (string, Keys, error) {
	key := mux.Vars(r)["key"]
	var keys Keys
	err := util.Decode(key, &keys)
	if err != nil {
		return key, keys, err
	}

	if keys.ID == nil {
		return key, keys, fmt.Errorf("keys id is required")
	}

	_, err = NewRepository(r.Context()).deleteById(*keys.ID)
	if err != nil {
		return key, keys, err
	}

	defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingID, keys.ReceiptID)

	return key, keys, nil
}
