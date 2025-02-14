package rates

import (
	"fmt"
	"github.com/a-h/templ"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"log"
	"net/http"
	"strings"
	"time"
)

const _PATH = "/api/rates"
const _SEARCH = _PATH + "/search"

func Routes(server *mux.Router) {
	server.Handle(_PATH, templ.Handler(Init())).Methods("GET")

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/currencies", loadCurrencies).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", deleteRate).Methods("DELETE")
}

func search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	currencies := query["currency_input"]
	rateQuery := RateQuery{
		LastId:     api.GetQueryParamAsInt(r, "next_page"),
		Limit:      31,
		DateOfRate: api.GetQueryParamAsDate(r, "date_input"),
		Currencies: currencies,
		SortOrder:  api.GetQueryParamAsSortOrderType(r, "sort_order"),
	}

	response, err := getRateTableResponse(rateQuery)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := response.Results

	var nextPageUrl string
	if len(results) == rateQuery.Limit {
		results = results[:len(results)-1]
		last := results[len(results)-1]
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%d", *last.Item.ID))
		if rateQuery.DateOfRate != nil {
			sb.WriteString(fmt.Sprintf("&date_input=%s", rateQuery.DateOfRate.Format(time.DateOnly)))
		}

		if len(rateQuery.Currencies) > 0 {
			for _, currency := range rateQuery.Currencies {
				sb.WriteString(fmt.Sprintf("&currency_input=%s", currency))
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

func loadCurrencies(w http.ResponseWriter, r *http.Request) {
	response, err := getCurrencies()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var builder strings.Builder
	builder.WriteString("currencies = [")
	for i, currency := range response {
		builder.WriteString(fmt.Sprintf("\"%s\"", currency))
		if i < len(response)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	err = Currencies(builder.String()).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteRate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var str int64
	err := api.Decode(id, &str)
	if err != nil {
		log.Printf("Error decoding id: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//number, err := strconv.ParseInt(str, 10, 64)
	//
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	query := r.URL.Query()
	currencies := query["currency_input"]
	rateQuery := RateQuery{
		DateOfRate: api.GetQueryParamAsDate(r, "date_input"),
		Currencies: currencies,
		SortOrder:  api.GetQueryParamAsSortOrderType(r, "sort_order"),
	}

	counters, err := deleteRateReturnCounters(str, rateQuery)

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
