package reserveFunds

import (
	"db/gen/model"
	"github.com/google/uuid"
)

type Type string

const (
	FIXED_PAY  Type = "FIXED_PAY"
	PERCENTAGE Type = "PERCENTAGE"
)

func IsFixedPay(str string) bool {
	return str == "FIXED_PAY"
}

type Item struct {
	CardId       string
	Key          string
	Item         model.ReserveFunds
	CreatedAt    int64
	UpdateParams *string
	isUpdate     *bool
}

const CardIdPrefix = "reserve-funds-"

func cardId() string {
	return CardIdPrefix + uuid.NewString()
}

type Keys struct {
	BuildingId string
	Id         *int32
	ReceiptId  *int64
	CardId     string
}

func keys(item model.ReserveFunds, cardId string) Keys {
	return Keys{
		BuildingId: item.BuildingID,
		Id:         item.ID,
		ReceiptId:  nil,
		CardId:     cardId,
	}
}

type FormDto struct {
	Key   string
	Items []Item
}

type UpdateParams struct {
	Key           string  `json:"key"`
	Name          string  `json:"name"`
	Fund          float64 `json:"fund"`
	Expense       float64 `json:"expense"`
	Pay           float64 `json:"pay"`
	Active        bool    `json:"active"`
	Type          string  `json:"type"`
	ExpenseType   string  `json:"expenseType"`
	AddToExpenses bool    `json:"addToExpenses"`
}
type FormResponse struct {
	errorStr string
	item     *Item
	counter  int64
}

type FormRequest struct {
	Key           string  `form:"key" validate:"required"`
	Name          string  `form:"name" validate:"required,min=3,max=100"`
	Fund          float64 `form:"fund"`
	Expense       float64 `form:"expense"`
	Pay           float64 `form:"pay" validate:"required,gt=0"`
	Active        bool    `form:"active"`
	Type          string  `form:"type" validate:"required,oneof=FIXED_PAY PERCENTAGE"`
	ExpenseType   string  `form:"expenseType" validate:"required,oneof=COMMON UNCOMMON"`
	AddToExpenses bool    `form:"addToExpenses"`
}
