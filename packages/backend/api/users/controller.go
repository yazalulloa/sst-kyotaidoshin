package users

import (
	"fmt"
	"github.com/gorilla/mux"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/users"
const _SEARCH = _PATH + "/search"

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", userDelete).Methods("DELETE")
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

	err = Search(response).Render(r.Context(), w)
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
