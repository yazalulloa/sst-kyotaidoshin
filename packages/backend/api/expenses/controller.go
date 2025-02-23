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

const _PATH = "/api/expenses"

func Routes(server *mux.Router) {

	server.HandleFunc(_PATH, expensesPut).Methods("PUT")
	server.HandleFunc(_PATH+"/{key}", expensesDelete).Methods("DELETE")
}

func expensesPut(w http.ResponseWriter, r *http.Request) {
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
			log.Printf("Error converting to item: %v", err)
			response.errorStr = err.Error()
			return response
		}

		var counter *int64

		if !isUpdate {
			count, err := countByReceipt(keys.ReceiptID)
			if err != nil {
				log.Printf("Error getting count: %v", err)
				response.errorStr = err.Error()
				return response
			}
			counter = &count
		}

		response.item = item
		response.item.isUpdate = &isUpdate
		response.counter = counter

		return response
	}

	response := upsert()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func expensesDelete(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	var keys Keys
	err := api.Decode(key, &keys)
	if err != nil {
		log.Printf("Error decoding key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if keys.ID == nil {
		log.Printf("Error deleting reserveFund: %v", "id is required")
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}

	_, err = deleteById(*keys.ID)
	if err != nil {
		log.Printf("Error deleting reserveFund: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := countByReceipt(keys.ReceiptID)

	if err != nil {
		log.Printf("Error getting count: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = DeleteResponse(count, key).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
