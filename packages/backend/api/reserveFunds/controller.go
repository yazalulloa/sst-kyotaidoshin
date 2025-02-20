package reserveFunds

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

const _PATH = "/api/reserveFunds"

func Routes(server *mux.Router) {

	server.HandleFunc(_PATH, reserveFundPut).Methods("PUT")
	server.HandleFunc(_PATH+"/{key}", reserveFundDelete).Methods("DELETE")
}

func reserveFundPut(w http.ResponseWriter, r *http.Request) {

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

		validate, _ := util.GetValidator()
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
			response.errorStr = err.Error()
			return response
		}

		item, err := getItem(idToLookup, cardIdStr)
		if err != nil {
			log.Printf("Error getting item: %v", err)
			response.errorStr = err.Error()
			return response
		}

		count, err := countByBuilding(keys.BuildingId)
		if err != nil {
			log.Printf("Error getting count: %v", err)
			response.errorStr = err.Error()
			return response
		}

		response.item = item
		response.item.isUpdate = &isUpdate
		response.counter = count

		return response
	}

	response := upsert()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func reserveFundDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	var keys Keys
	err := api.Decode(key, &keys)
	if err != nil {
		log.Printf("Error decoding key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if keys.Id == nil {
		log.Printf("Error deleting reserveFund: %v", "id is required")
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}

	_, err = deleteById(*keys.Id)
	if err != nil {
		log.Printf("Error deleting reserveFund: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := countByBuilding(keys.BuildingId)

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
