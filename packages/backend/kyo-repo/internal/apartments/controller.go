package apartments

import (
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/isr"
	"github.com/yaz/kyo-repo/internal/receiptPdf"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/apartments"
const _SEARCH = _PATH + "/search"
const _UPLOAD_BACKUP_FORM = _PATH + "/uploadBackupForm"
const _UPLOAD_BACKUP = _PATH + "/upload/backup"

func Routes(holder *api.RouterHolder) {

	holder.GET(_SEARCH, search, api.APARTMENTS_READ)
	holder.PUT(_PATH, aptPut, api.ApartmentsUpsertRecaptchaAction, api.APARTMENTS_WRITE)
	holder.DELETE(_PATH+"/{key}", aptDelete, api.ApartmentsDeleteRecaptchaAction, api.APARTMENTS_WRITE)
	holder.GET(_UPLOAD_BACKUP_FORM, getUploadBackupForm, api.APARTMENTS_UPLOAD_BACKUP)
	holder.POST(_UPLOAD_BACKUP, uploadBackup, api.ApartmentsUploadBackupRecaptchaAction, api.APARTMENTS_UPLOAD_BACKUP)
}

func search(w http.ResponseWriter, r *http.Request) {
	nextPage := util.GetQueryParamAsString(r, "next_page")
	var keys Keys
	if nextPage != "" {
		err := util.Decode(nextPage, &keys)

		if err != nil {
			log.Printf("failed to decode nextPage: %v", err)
			http.Error(w, "Bad Request nextPage", http.StatusBadRequest)
			return
		}
	}
	query := r.URL.Query()
	buildings := query["building_input"]
	requestQuery := RequestQuery{
		lastBuildingId: keys.BuildingId,
		lastNumber:     keys.Number,
		q:              util.GetQueryParamAsString(r, "apt_search_input"),
		buildings:      buildings,
		Limit:          31,
	}

	response, err := NewService(r.Context()).getTableResponse(requestQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := response.Results
	var nextPageUrl string
	if len(results) == requestQuery.Limit {
		results = results[:len(results)-1]
		last := results[len(results)-1]
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%s", last.Key))

		if requestQuery.q != "" {
			sb.WriteString(fmt.Sprintf("&apt_search_input=%s", requestQuery.q))
		}

		if len(requestQuery.buildings) > 0 {
			for _, building := range requestQuery.buildings {
				sb.WriteString(fmt.Sprintf("&building_input=%s", building))
			}
		}

		nextPageUrl = sb.String()
	}

	response.NextPageUrl = nextPageUrl
	response.Results = results

	err = Search(*response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func aptDelete(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	counters, err := NewService(r.Context()).deleteAndReturnCounters(keys)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingId)
	defer isr.Invoke(r.Context())

	err = CountersView(*counters).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// TODO when creating a new apartment create new debt for every receipt
func aptPut(w http.ResponseWriter, r *http.Request) {
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

		isUpdate := request.Key != ""

		var keys Keys

		if isUpdate {
			err = util.Decode(request.Key, &keys)
			if err != nil {
				log.Printf("Error decoding key: %v", err)
				response.errorStr = err.Error()
				return response
			}
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

		service := NewService(r.Context())

		if !isUpdate {
			exists, err := service.repo.aptExists(request.Building, request.Number)
			if err != nil {
				response.errorStr = err.Error()
				return response
			}

			if exists {
				response.errorStr = fmt.Sprintf("Apartment %s already exists", request.Number)
				return response
			}
		}
		// todo improve this
		//if !isUpdate {
		//	buildingIds, err := buildingIds()
		//	if err != nil {
		//		response.errorStr = err.Error()
		//		return response
		//	}
		//
		//	if !slices.Contains(buildingIds, request.Building) {
		//		response.errorStr = fmt.Sprintf("Building ID %s does not exist", request.Building)
		//		return response
		//	}
		//}

		apartment := model.Apartments{
			BuildingID: request.Building,
			Number:     request.Number,
			Name:       request.Name,
			Aliquot:    request.Aliquot,
			Emails:     strings.Join(request.Emails, ","),
		}

		if isUpdate {
			apartment.BuildingID = keys.BuildingId
			apartment.Number = keys.Number

			err = service.repo.update(apartment)
		} else {
			err = service.repo.insert(apartment)
		}

		if err != nil {
			log.Printf("Error inserting/updating reserveFund: %v", err)
			response.errorStr = err.Error()
			return response
		}

		defer receiptPdf.PublishBuilding(r.Context(), keys.BuildingId)
		defer isr.Invoke(r.Context())

		if isUpdate {
			item, err := toItem(&apartment, &keys.CardId)
			if err != nil {
				log.Printf("Error getting item: %v", err)
				response.errorStr = err.Error()
				return response
			}

			item.isUpdate = true
			response.item = item
		}

		return response
	}

	response := upsert()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getUploadBackupForm(w http.ResponseWriter, r *http.Request) {

	component, err := api.BuildUploadForm(r, "BACKUPS/APARTMENTS/")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadBackup(w http.ResponseWriter, r *http.Request) {

	component, err := api.ProcessUploadBackup(r, "/apartments", NewService(r.Context()).ProcessDecoder)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
