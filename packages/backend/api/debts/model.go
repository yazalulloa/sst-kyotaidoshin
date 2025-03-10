package debts

import (
	"db/gen/model"
	"github.com/google/uuid"
)

type DebtDto struct {
	BuildingID                    string  `json:"building_id"`
	ReceiptID                     int64   `json:"receipt_id"`
	AptNumber                     string  `json:"apt_number"`
	Receipts                      int16   `json:"receipt"`
	Amount                        float64 `json:"amount"`
	Months                        []int16 `json:"months"`
	PreviousPaymentAmount         float64 `json:"previous_payment_amount"`
	PreviousPaymentAmountCurrency string  `json:"previous_payment_amount_currency"`
}

type MonthlyDebt struct {
	Amount int              `json:"amount"`
	Years  []YearWithMonths `json:"years"`
}

type YearWithMonths struct {
	Year   int16   `json:"year" validate:"gte=2020,lte=2100"`
	Months []int16 `json:"months" validate:"gte=1,lte-12,dive,required,gte=1,lte=12"`
}

type FormDto struct {
	Items  []Item
	Totals Totals
}

type Totals struct {
	TotalAmount   float64
	TotalReceipts int16
	Counter       int
}

type Item struct {
	CardId       string
	Key          string
	Item         model.Debts
	Months       MonthlyDebt
	UpdateParams *string
	isUpdate     *bool
}

func cardId() string {
	return "debts-" + uuid.NewString()
}

type Keys struct {
	BuildingID string
	ReceiptID  string
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
	Key                           string      `json:"key"`
	Apt                           string      `json:"apt"`
	Receipts                      int16       `json:"receipts"`
	Amount                        float64     `json:"amount"`
	Months                        MonthlyDebt `json:"months"`
	PreviousPaymentAmount         float64     `json:"previous_payment_amount"`
	PreviousPaymentAmountCurrency string      `json:"previous_payment_amount_currency"`
}
type FormResponse struct {
	errorStr string
	item     *Item
	Totals   *Totals
}

type FormRequest struct {
	Key                           string   `form:"key" validate:"required,notblank,max=300"`
	Receipts                      int16    `form:"receipts" validate:"gte=0"`
	Amount                        float64  `form:"amount" validate:"gte=0"`
	DebtMonthsTotal               int      `form:"debtMonthsTotal" validate:"gte=0"`
	DebtMonths                    []string `form:"debtMonths" validate:""`
	PreviousPaymentAmount         float64  `form:"previousPaymentAmount" validate:"gte=0"`
	PreviousPaymentAmountCurrency string   `form:"previousPaymentAmountCurrency" validate:"required,oneof=USD VED"`
}
