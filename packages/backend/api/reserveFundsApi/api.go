package reserveFundsApi

import (
	"context"
	"github.com/a-h/templ"
	"github.com/gorilla/mux"
	"kyotaidoshin/expenses"
	"kyotaidoshin/receipts"
	"kyotaidoshin/reserveFunds"
	"log"
	"net/http"
)

func Routes(server *mux.Router) {

	server.HandleFunc(reserveFunds.PATH, reserveFundPut).Methods("PUT")
	server.HandleFunc(reserveFunds.PATH+"/{key}", reserveFundDelete).Methods("DELETE")
}

func reserveFundPut(w http.ResponseWriter, r *http.Request) {

	response := reserveFunds.Upsert(r)

	var ctx context.Context
	if response.Keys.ReceiptId != "" && response.ErrorStr == "" {
		expensesDto, err := receipts.JoinExpensesAndReserveFunds(response.Item.Item.BuildingID, response.Keys.ReceiptId)
		if err != nil {
			log.Printf("Error joining expenses and reserve funds: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		expensesDto.IsTherePercentage = true
		ctx = templ.WithChildren(r.Context(), expenses.DeleteResponse("", *expensesDto))
	} else {
		ctx = r.Context()
	}

	err := reserveFunds.FormResponseView(response).Render(ctx, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func reserveFundDelete(w http.ResponseWriter, r *http.Request) {

	key, keys, err := reserveFunds.DeleteAndReturnKeys(r)

	if err != nil {
		log.Printf("Error deleting reserve fund: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ctx context.Context

	if keys.ReceiptId != "" {
		expensesDto, err := receipts.JoinExpensesAndReserveFunds(keys.BuildingId, keys.ReceiptId)
		if err != nil {
			log.Printf("Error joining expenses and reserve funds: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		expensesDto.IsTherePercentage = true
		ctx = templ.WithChildren(r.Context(), expenses.DeleteResponse(key, *expensesDto))
	} else {
		ctx = r.Context()
	}

	counter, err := reserveFunds.CountByBuilding(keys.BuildingId)
	if err != nil {
		log.Printf("Error getting count: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = reserveFunds.DeleteResponse(counter, key).Render(ctx, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
