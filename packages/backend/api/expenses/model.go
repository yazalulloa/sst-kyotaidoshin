package expenses

import (
	"db/gen/model"
	"github.com/google/uuid"
)

type ExpenseDto struct {
	BuildingID  string  `json:"building_id"`
	ReceiptID   int32   `json:"receipt_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	ReserveFund bool    `json:"reserve_fund"`
	Type        string  `json:"type"`
}

type FormDto struct {
	Key   string
	Items []Item
}

type Item struct {
	CardId       string
	Key          string
	Item         model.Expenses
	UpdateParams *string
	isUpdate     *bool
}

func cardId() string {
	return "expenses-" + uuid.NewString()
}

type Keys struct {
	ID         *int32
	BuildingID string
	ReceiptID  int32
	CardId     string
}

func keys(item model.Expenses, cardId string) Keys {
	return Keys{
		ID:         item.ID,
		BuildingID: item.BuildingID,
		ReceiptID:  item.ReceiptID,
		CardId:     cardId,
	}
}

type UpdateParams struct {
	Key         string  `json:"key"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Type        string  `json:"type"`
}
type FormResponse struct {
	errorStr string
	item     *Item
	counter  int64
}
