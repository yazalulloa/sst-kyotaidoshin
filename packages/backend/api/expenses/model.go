package expenses

type ExpenseDto struct {
	BuildingID  string  `json:"building_id"`
	ReceiptID   int32   `json:"receipt_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	ReserveFund bool    `json:"reserve_fund"`
	Type        string  `json:"type"`
}
