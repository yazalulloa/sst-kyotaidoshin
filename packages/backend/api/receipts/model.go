package receipts

import (
	"db/gen/model"
	"github.com/google/uuid"
	"golang.org/x/sync/syncmap"
	"kyotaidoshin/api"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/util"
	"time"
)

type RequestQuery struct {
	LastId    int32
	Limit     int64
	Buildings []string
	Months    []int16
	Years     []int16
	SortOrder util.SortOrderType
}

type TableResponse struct {
	Counters    Counters
	NextPageUrl string
	Results     []Item
}

type Counters struct {
	TotalCount int64
	QueryCount *int64
}

type Item struct {
	CardId       string
	Key          string
	Item         model.Receipts
	CreatedAt    int64
	LastSent     *int64
	UpdateParams *string
	isUpdate     bool
}

func cardId() string {
	return "receipts-" + uuid.NewString()
}

type Keys struct {
	BuildingId string
	Id         int32
	CardId     string
}

func keys(receipt model.Receipts, cardId string) Keys {
	return Keys{
		BuildingId: receipt.BuildingID,
		Id:         *receipt.ID,
		CardId:     cardId,
	}
}

type InitDto struct {
	BuildingIds        string
	UploadBackupParams *api.UploadBackupParams
	TableResponse      TableResponse
}

type ReceiptRecord struct {
	Receipt      ReceiptDto                    `json:"receipt"`
	ExtraCharges []extraCharges.ExtraChargeDto `json:"extra_charges"`
	Expenses     []expenses.ExpenseDto         `json:"expenses"`
	Debts        []debts.DebtDto               `json:"debts"`
}

type ReceiptDto struct {
	BuildingID string  `json:"building_id"`
	Year       int16   `json:"year"`
	Month      int16   `json:"month"`
	Date       string  `json:"date"`
	Sent       bool    `json:"sent"`
	LastSent   *string `json:"last_sent"`
}

type RatesHolder struct {
	Rates syncmap.Map
}

type UpdateParams struct {
	Key    string
	Year   int16
	Month  int16
	Date   time.Time
	RateID int64
}
