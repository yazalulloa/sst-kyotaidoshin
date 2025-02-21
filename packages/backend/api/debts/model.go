package debts

type DebtDto struct {
	BuildingID                    string  `json:"building_id"`
	ReceiptID                     int64   `json:"receipt_id"`
	AptNumber                     string  `json:"apt_number"`
	Receipts                      int16   `json:"receipts"`
	Amount                        float64 `json:"amount"`
	Months                        []int32 `json:"months"`
	PreviousPaymentAmount         float64 `json:"previous_payment_amount"`
	PreviousPaymentAmountCurrency string  `json:"previous_payment_amount_currency"`
}
