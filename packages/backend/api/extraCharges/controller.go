package extraCharges

import (
	"db/gen/model"
	"fmt"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"kyotaidoshin/receiptPdf"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/extraCharges"

func Routes(holder *api.RouterHolder) {

	holder.PUT(_PATH, extraChargesPut, api.RECEIPTS_WRITE, api.BUILDINGS_WRITE)
	holder.DELETE(_PATH+"/{key}", extraChargesDelete, api.RECEIPTS_WRITE, api.BUILDINGS_WRITE)
}

func extraChargesPut(w http.ResponseWriter, r *http.Request) {
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

		apartments := strings.Join(request.Apartments, ",")
		extraCharge := model.ExtraCharges{
			ID:              keys.Id,
			BuildingID:      keys.BuildingID,
			ParentReference: keys.ParentReference,
			Type:            keys.Type,
			Description:     request.Description,
			Amount:          request.Amount,
			Currency:        request.Currency,
			Active:          request.Active,
			Apartments:      apartments,
		}

		isUpdate := keys.Id != nil

		var idToLookup int32
		var cardIdStr *string
		if isUpdate {
			_, err = update(extraCharge)
			idToLookup = *keys.Id
			cardIdStr = &keys.CardId
		} else {
			lastInsertId, err := insert(extraCharge)
			if err == nil {
				idToLookup = int32(lastInsertId)
			}
		}

		if err != nil {
			log.Printf("Error inserting/updating extraCharge: %v", err)
			response.errorStr = err.Error()
			return response
		}

		if keys.Type == TypeReceipt {
			defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingID, keys.ParentReference)
		} else {
			defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingID)
		}

		item, err := getItem(idToLookup, cardIdStr)
		if err != nil {
			log.Printf("Error getting item: %v", err)
			response.errorStr = err.Error()
			return response
		}

		count, err := countByBuilding(keys.BuildingID)
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

func extraChargesDelete(w http.ResponseWriter, r *http.Request) {

	key := mux.Vars(r)["key"]
	var keys Keys
	err := util.Decode(key, &keys)
	if err != nil {
		log.Printf("Error decoding key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if keys.Id == nil {
		log.Printf("Error deleting extraCharges: %v", "id is required")
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}

	_, err = deleteById(*keys.Id)
	if err != nil {
		log.Printf("Error deleting extraCharges: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, err := countByBuilding(keys.BuildingID)

	if err != nil {
		log.Printf("Error getting count: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if keys.Type == TypeReceipt {
		defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingID, keys.ParentReference)
	} else {
		defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingID)
	}

	err = DeleteResponse(count, key).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
