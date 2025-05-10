package reserveFundsApi

import (
	"context"
	"github.com/a-h/templ"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/expenses"
	"github.com/yaz/kyo-repo/internal/receipts"
	"github.com/yaz/kyo-repo/internal/reserveFunds"
	"log"
	"net/http"
)

func Routes(holder *api.RouterHolder) {

	holder.PUT(reserveFunds.PATH, reserveFundPut, api.ReserveFundsUpsertRecaptchaAction, api.RECEIPTS_WRITE, api.BUILDINGS_WRITE)
	holder.DELETE(reserveFunds.PATH+"/{key}", reserveFundDelete, api.ReserveFundsDeleteRecaptchaAction, api.RECEIPTS_WRITE, api.BUILDINGS_WRITE)
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
