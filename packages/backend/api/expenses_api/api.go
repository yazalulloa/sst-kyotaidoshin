package expenses_api

import (
	"github.com/gorilla/mux"
	"kyotaidoshin/expenses"
	"kyotaidoshin/receipts"
	"log"
	"net/http"
)

func Routes(server *mux.Router) {

	server.HandleFunc(expenses.PATH, expensesPut).Methods("PUT")
	server.HandleFunc(expenses.PATH+"/{key}", expensesDelete).Methods("DELETE")
}

func expensesPut(w http.ResponseWriter, r *http.Request) {
	response := expenses.Upsert(r)

	if response.ErrorStr == "" {

		expensesDto, err := receipts.JoinExpensesAndReserveFunds(response.Item.Item.BuildingID, response.Item.Item.ReceiptID)
		if err != nil {
			log.Printf("Error joining expenses and reserve funds: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response.ReceiptExpensesDto = expensesDto
	}

	err := expenses.FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func expensesDelete(w http.ResponseWriter, r *http.Request) {
	keys, err := expenses.DeleteAndReturnKeys(r)
	if err != nil {
		log.Printf("Error deleting expense: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expensesDto, err := receipts.JoinExpensesAndReserveFunds(keys.BuildingID, keys.ReceiptID)

	if err != nil {
		log.Printf("Error joining expenses and reserve funds: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = expenses.DeleteResponse(*expensesDto).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
