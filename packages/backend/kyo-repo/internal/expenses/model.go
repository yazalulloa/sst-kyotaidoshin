package expenses

import (
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
)

const AliquotDifference = "DIFERENCIA DE ALIQUOTA"

type ExpenseType string

const (
	COMMON   ExpenseType = "COMMON"
	UNCOMMON ExpenseType = "UNCOMMON"
)

func (receiver ExpenseType) Name() string {
	return string(receiver)
}

func (receiver ExpenseType) Is(str string) bool {
	return receiver.Name() == str
}

func (receiver ExpenseType) ExpenseIs(item model.Expenses) bool {
	return receiver.Is(item.Type)
}

type ExpenseDto struct {
	BuildingID  string  `json:"building_id"`
	ReceiptID   int32   `json:"receipt_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	ReserveFund bool    `json:"reserve_fund"`
	Type        string  `json:"type"`
}

type ExpenseBackup struct {
	BuildingID  string  `json:"building_id"`
	ReceiptID   string  `json:"receipt_id"`
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

const CardIdPrefix = "expenses-"

func CardId() string {
	return CardIdPrefix + uuid.NewString()
}

type Keys struct {
	ID         *int32
	BuildingID string
	ReceiptID  string
	CardId     *string
}

func keys(item model.Expenses, cardId *string) Keys {
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
	ErrorStr           string
	Item               *Item
	counter            *int64
	ReceiptExpensesDto *ReceiptExpensesDto
}

type FormRequest struct {
	Key         string  `form:"key" validate:"required,notblank,max=300"`
	Description string  `form:"description" validate:"required,notblank,max=100"`
	Amount      float64 `form:"amount" validate:"required,ne=0"`
	Currency    string  `form:"currency" validate:"required,oneof=USD VED"`
	Type        string  `form:"type" validate:"required,oneof=COMMON UNCOMMON"`
}

type ExpenseTotals struct {
	ExpensesCounter          int
	TotalCommon              float64
	TotalUnCommon            float64
	TotalCommonPlusReserve   float64
	TotalUnCommonPlusReserve float64
}

type ReceiptExpensesDto struct {
	IsTherePercentage   bool
	ReserveFundExpenses []Item
	Totals              ExpenseTotals
}
