package rates

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/util"
)

const _PATH = "/api/rates"
const _SEARCH = _PATH + "/search"

func Routes(holder *api.RouterHolder) {

	holder.GET(_SEARCH, search, api.RATES_READ)
	holder.DELETE(_PATH+"/{id}", deleteRate, api.RatesDeleteRecaptchaAction, api.RATES_WRITE)
	holder.PUT(_PATH+"/trend", updateTrend, api.RatesDeleteRecaptchaAction, api.RATES_WRITE)
}

func search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	currencies := query["currency_input"]
	requestQuery := RequestQuery{
		LastId:     util.GetQueryParamAsInt(r, "next_page"),
		Limit:      31,
		DateOfRate: util.GetQueryParamAsDate(r, "date_input"),
		Currencies: currencies,
		SortOrder:  util.GetQueryParamAsSortOrderType(r, "sort_order"),
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
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%d", *last.Item.ID))
		if requestQuery.DateOfRate != nil {
			sb.WriteString(fmt.Sprintf("&date_input=%s", requestQuery.DateOfRate.Format(time.DateOnly)))
		}

		if len(requestQuery.Currencies) > 0 {
			for _, currency := range requestQuery.Currencies {
				sb.WriteString(fmt.Sprintf("&currency_input=%s", currency))
			}
		}

		sb.WriteString(fmt.Sprintf("&sort_order=%s", requestQuery.SortOrder))

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

func deleteRate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var str int64
	err := util.Decode(id, &str)
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
	rateQuery := RequestQuery{
		DateOfRate: util.GetQueryParamAsDate(r, "date_input"),
		Currencies: currencies,
		SortOrder:  util.GetQueryParamAsSortOrderType(r, "sort_order"),
	}

	counters, err := NewService(r.Context()).deleteRateReturnCounters(str, rateQuery)

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

func updateTrend(w http.ResponseWriter, r *http.Request) {

	err := NewService(r.Context()).UpdateStableTrend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
