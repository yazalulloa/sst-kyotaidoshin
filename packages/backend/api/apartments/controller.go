package apartments

import (
	"fmt"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/apartments"
const _SEARCH = _PATH + "/search"

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{key}", aptDelete).Methods("DELETE")
	//server.HandleFunc(_PATH, aptPut).Methods("PUT")
	//server.HandleFunc(_PATH+"/formData", formData).Methods("GET")
}

func search(w http.ResponseWriter, r *http.Request) {
	nextPage := api.GetQueryParamAsString(r, "next_page")
	var keys Keys
	if nextPage != "" {
		err := api.Decode(nextPage, &keys)

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
		q:              api.GetQueryParamAsString(r, "q"),
		buildings:      buildings,
		Limit:          31,
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

		if requestQuery.q != "" {
			sb.WriteString(fmt.Sprintf("&q=%s", requestQuery.q))
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

	err = Search(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func aptDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyStr := vars["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := api.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode nextPage: %v", err)
		http.Error(w, "Bad Request nextPage", http.StatusBadRequest)
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
