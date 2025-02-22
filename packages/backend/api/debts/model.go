package debts

import (
	"db/gen/model"
	"github.com/google/uuid"
)

type DebtDto struct {
	BuildingID                    string  `json:"building_id"`
	ReceiptID                     int64   `json:"receipt_id"`
	AptNumber                     string  `json:"apt_number"`
	Receipts                      int16   `json:"receipts"`
	Amount                        float64 `json:"amount"`
	Months                        []int16 `json:"months"`
	PreviousPaymentAmount         float64 `json:"previous_payment_amount"`
	PreviousPaymentAmountCurrency string  `json:"previous_payment_amount_currency"`
}

type FormDto struct {
	Items []Item
}

type Item struct {
	CardId       string
	Key          string
	Item         model.Debts
	UpdateParams *string
}

func cardId() string {
	return "debts-" + uuid.NewString()
}

type Keys struct {
	BuildingID string
	ReceiptID  int32
	AptNumber  string
	CardId     string
}

func keys(item model.Debts, cardId string) Keys {
	return Keys{
		BuildingID: item.BuildingID,
		ReceiptID:  item.ReceiptID,
		AptNumber:  item.AptNumber,
		CardId:     cardId,
	}
}

type UpdateParams struct {
	Key                           string  `json:"key"`
	Receipts                      int16   `json:"receipts"`
	Amount                        float64 `json:"amount"`
	Months                        []int16 `json:"months"`
	PreviousPaymentAmount         float64 `json:"previous_payment_amount"`
	PreviousPaymentAmountCurrency string  `json:"previous_payment_amount_currency"`
}
type FormResponse struct {
	errorStr string
	item     *Item
	counter  int64
}
