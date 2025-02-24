package expenses

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
	err = api.Decode(request.Key, &keys)
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

	if isUpdate {
		_, err = update(expense)
	} else {
		lastInsertId, err := insert(expense)
		if err == nil {
			id := int32(lastInsertId)
			keys.ID = &id
		}
	}

	item, err := toItem(&expense, keys.CardId)
	if err != nil {
		log.Printf("Error converting to Item: %v", err)
		response.ErrorStr = err.Error()
		return response
	}

	var counter *int64

	if !isUpdate {
		count, err := countByReceipt(keys.ReceiptID)
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
	err := api.Decode(key, &keys)
	if err != nil {
		return key, keys, err
	}

	if keys.ID == nil {
		return key, keys, fmt.Errorf("id is required")
	}

	_, err = deleteById(*keys.ID)
	if err != nil {
		return key, keys, err
	}

	return key, keys, nil
}
