package roles

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
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

const _PATH = "/api/roles"
const _SEARCH = _PATH + "/search"

func Routes(holder *api.RouterHolder) {

	holder.GET(_SEARCH, search, api.ROLES_READ)
	holder.PUT(_PATH, rolesPut, api.RolesUpsertRecaptchaAction, api.ROLES_WRITE)
	holder.DELETE(_PATH+"/{key}", roleDelete, api.RolesDeleteRecaptchaAction, api.ROLES_WRITE)
	holder.GET(_PATH+"/all", getAll, api.ROLES_READ)
	holder.GET(_PATH+"/all/min", getAllMin, api.ROLES_READ)
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

	requestQuery := RequestQuery{
		LastId: keys.ID,
		Q:      util.GetQueryParamAsString(r, "q"),
		Limit:  31,
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
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%s", last.Key))

		if requestQuery.Q != "" {
			sb.WriteString(fmt.Sprintf("&q=%s", requestQuery.Q))
		}

		nextPageUrl = sb.String()
	}

	response.NextPageUrl = nextPageUrl
	response.Results = results

	err = searchView(*response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func rolesPut(w http.ResponseWriter, r *http.Request) {
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

		role := model.Roles{
			Name: strings.TrimSpace(request.Name),
		}

		if isUpdate {
			role.ID = &keys.ID
			_, err = updateRole(role, request.Perms)
		} else {
			roleId, err := insertRole(request.Name, request.Perms)

			log.Printf("roleId: %v", roleId)
			if err == nil {
				role.ID = &roleId
			}

		}

		if err != nil {
			log.Printf("Error inserting/updating role: %v", err)
			response.errorStr = err.Error()
			return response
		}

		if isUpdate {
			item, err := getItem(*role.ID, &keys.CardId)
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

func roleDelete(w http.ResponseWriter, r *http.Request) {
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

	counters, err := deleteAndReturnCounters(keys)

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

func getAll(w http.ResponseWriter, r *http.Request) {

	data, err := selectAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	roles := make([]RoleWithPermissions, len(data))

	for i, item := range data {
		roles[i] = RoleWithPermissions{
			Role:        item.Roles,
			Permissions: item.Permissions,
		}
	}

	byteArray, err := json.Marshal(roles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(byteArray)

	err = AllData(base64Str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getAllMin(w http.ResponseWriter, r *http.Request) {
	data, err := selectAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	roles := make([]RoleMin, len(data))

	for i, item := range data {
		roles[i] = RoleMin{
			ID:          *item.Roles.ID,
			Name:        item.Roles.Name,
			PermsLength: len(item.Permissions),
		}
	}

	byteArray, err := json.Marshal(roles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(byteArray)

	err = AllData(base64Str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
