package receipts

import (
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/debts"
	"github.com/yaz/kyo-repo/internal/expenses"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/reserveFunds"
	"github.com/yaz/kyo-repo/internal/util"
	"golang.org/x/sync/syncmap"
	"time"
)

type RequestQuery struct {
	LastId    string
	Limit     int64
	Buildings []string
	Months    []int16
	Years     []int16
	Date      *time.Time
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
	UpdateParams string
	isUpdate     bool
}

func cardId() string {
	return "receipts-" + uuid.NewString()
}

type Keys struct {
	BuildingId string
	Id         string
	CardId     string
}

func keys(receipt model.Receipts, cardId string) Keys {
	return Keys{
		BuildingId: receipt.BuildingID,
		Id:         receipt.ID,
		CardId:     cardId,
	}
}

type DownloadKeys struct {
	BuildingId string
	Id         string
	Date       string
	Year       int16
	Month      int16
	Elem       string
	Parts      []string
	AllApt     bool
}

type InitDto struct {
	BuildingIds        string
	UploadBackupParams *util.UploadBackupParams
	TableResponse      TableResponse
}

type ReceiptRecord struct {
	Receipt      ReceiptDto                    `json:"receipt"`
	ExtraCharges []extraCharges.ExtraChargeDto `json:"extra_charges"`
	Expenses     []expenses.ExpenseDto         `json:"expenses"`
	Debts        []debts.DebtDto               `json:"debts"`
}

type ReceiptBackup struct {
	Receipt      ReceiptDto                    `json:"receipt"`
	ExtraCharges []extraCharges.ExtraChargeDto `json:"extra_charges"`
	Expenses     []expenses.ExpenseBackup      `json:"expenses"`
	Debts        []debts.DebtBackup            `json:"debts"`
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
	Key      string `json:"key"`
	Building string `json:"building"`
	Year     int16  `json:"year"`
	Month    int16  `json:"month"`
	Date     string `json:"date"`
}

type RateDto struct {
	ID         int64
	Key        string
	Rate       float64
	DateOfRate string
}

type FormDto struct {
	key                 string
	updateParams        string
	building            model.Buildings
	receipt             *model.Receipts
	rates               []RateDto
	expenseFormDto      expenses.FormDto
	reserveFundExpenses []expenses.Item
	reserveFundFormDto  reserveFunds.FormDto
	extraChargesFormDto extraCharges.FormDto
	debtFormDto         debts.FormDto
	expenseTotals       expenses.ExpenseTotals
	apts                string
}
type FormRequest struct {
	Key     string `form:"key" validate:"required,min=3,max=200"`
	Year    int16  `form:"year" validate:"required,gt=2015,lte=2100"`
	Month   int16  `form:"month" validate:"required,gte=1,lte=12"`
	Date    string `form:"date" validate:"required,min=10,max=10"`
	RateKey string `form:"rate" validate:"required,min=3,max=100"`
}

type FormResponse struct {
	Key      *string
	errorStr string
}

type CalculatedReceipt struct {
	Expenses                      []ExpenseAttr
	TotalCommonExpenses           float64
	TotalCommonExpensesCurrency   util.AllowedCurrencies
	TotalUnCommonExpenses         float64
	TotalUnCommonExpensesCurrency util.AllowedCurrencies
	Apartments                    []AptTotal
	ApartmentsTotal               float64
	DebtReceiptsAmount            int16
	DebtTotals                    []AmountWithCurrency
	Rate                          model.Rates
	ReserveFunds                  []ReserveFundWithCalculatedAmount
	ThereIsReserveFundExpense     bool
	Building                      model.Buildings
	CurrenciesToShowAmountToPay   []util.AllowedCurrencies
	DebtsCurrenciesToShow         []util.AllowedCurrencies
	Receipt                       model.Receipts
	MonthStr                      string
	BuildingDownloadKeys          string
}

type AptTotal struct {
	Apartment    model.Apartments
	Amounts      []AmountWithCurrency
	ExtraCharges []model.ExtraCharges
	Debt         DebtTotal
	DebtMonthStr string
	DownloadKeys string
}

type DebtTotal struct {
	Debt    model.Debts
	Amounts []AmountWithCurrency
}

type ReserveFundWithCalculatedAmount struct {
	Fund   model.ReserveFunds
	Amount float64

	FundFormatted    string
	ExpenseFormatted string
	AmountToPay      string
	NewReserveFund   string
}

type ExpenseAttr struct {
	Expense       model.Expenses
	IsReserveFund bool
}

type TabId struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ReceiptFileFormDto struct {
	Month     int16          `json:"month"`
	Year      int16          `json:"year"`
	Years     []int16        `json:"years"`
	Building  string         `json:"building"`
	Buildings []string       `json:"buildings"`
	Filename  string         `json:"filename"`
	Date      string         `json:"date"`
	Rates     []rates.Option `json:"rates"`
	Data      string         `json:"data"`
}

type ReceiptNewFormRequest struct {
	Month    int16  `form:"month" validate:"required,gte=1,lte=12"`
	Year     int16  `form:"year" validate:"required,gte=2020,lte=2100"`
	Building string `form:"building" validate:"required,notblank,min=3,max=100"`
	Date     string `form:"date" validate:"required,notblank,min=10,max=10"`
	Rate     string `form:"rate" validate:"required,notblank,min=3,max=20"`
	Data     string `form:"data" validate:"required,notblank,min=3,max=10000"`
}

type ReceiptFull struct {
	model.Receipts
	Expenses     []model.Expenses
	Debts        []model.Debts
	ExtraCharges []model.ExtraCharges
}

type SendFormRequest struct {
	Key        string   `form:"key" validate:"required,notblank,min=3,max=200"`
	Subject    string   `form:"subject" validate:"required,min=3,max=200"`
	Message    string   `form:"message" validate:"required,min=3,max=1000"`
	Apartments []string `form:"apt_input" validate:"required,gte=1,lte=100,dive,notblank"`
}
