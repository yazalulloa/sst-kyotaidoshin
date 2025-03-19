package users

import (
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

const _PATH = "/api/users"
const _SEARCH = _PATH + "/search"

func Routes(holder *api.RouterHolder) {

	holder.GET(_SEARCH, search, api.USERS_READ)
	holder.DELETE(_PATH+"/{id}", userDelete, api.USERS_WRITE)
	holder.PUT(_PATH+"/role", userRolePatch, api.USERS_WRITE)
}

func search(w http.ResponseWriter, r *http.Request) {

	requestQuery := RequestQuery{
		LastId:    util.GetQueryParamAsString(r, "next_page"),
		Limit:     31,
		SortOrder: util.GetQueryParamAsSortOrderType(r, "sort_order"),
	}

	response, err := getTableResponse(requestQuery)

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
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%s", last.Item.ID))

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

func userDelete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var str string
	err := util.Decode(id, &str)
	if err != nil {
		log.Printf("Error decoding id: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if str == "" {
		http.Error(w, "Bad Request id", http.StatusBadRequest)
		return
	}

	counters, err := deleteRateReturnCounters(str, RequestQuery{})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = CountersView(*counters).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func userRolePatch(w http.ResponseWriter, r *http.Request) {

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

		if request.Key == "" {
			response.errorStr = "Bad Request"
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

		if request.RoleId < 0 {
			response.errorStr = "Bad Request"
			return response
		}

		var roleId *int32
		if request.RoleId > 0 {
			roleId = &request.RoleId
		}

		rowsAffected, err := updateRole(keys.ID, roleId)

		if err != nil {
			response.errorStr = err.Error()
			return response
		}

		log.Printf("Updated userRole: %d", rowsAffected)

		item, err := getItemWitRole(keys)

		if err != nil {
			response.errorStr = err.Error()
			return response
		}

		item.isUpdate = true
		response.item = item

		return response
	}

	response := upsert()

	err := UserRoleFormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
