package receipts

import (
	"db/gen/model"
	"github.com/google/uuid"
	"golang.org/x/sync/syncmap"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
)

type RequestQuery struct {
	LastId    string
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
	Part       string
	IsApt      bool
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
	Key   string `json:"key"`
	Year  int16  `json:"year"`
	Month int16  `json:"month"`
	Date  string `json:"date"`
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
	Key     string `form:"key" validate:"required,min=3,max=100"`
	Year    int16  `form:"year" validate:"required,gt=2015,lte=2100"`
	Month   int16  `form:"month" validate:"required,gte=1,lte=12"`
	Date    string `form:"date" validate:"required,min=10,max=10"`
	RateKey string `form:"rate" validate:"required,min=3,max=100"`
}

type FormResponse struct {
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
	DebtTotal                     float64
	Rate                          model.Rates
	ReserveFunds                  []ReserveFundWithCalculatedAmount
	ThereIsReserveFundExpense     bool
	Building                      model.Buildings
	CurrenciesToShowAmountToPay   []util.AllowedCurrencies
	Receipt                       model.Receipts
	MonthStr                      string
	BuildingDownloadKeys          string
	Key                           string
}

type AptTotal struct {
	Apartment    model.Apartments
	Amounts      []AmountWithCurrency
	ExtraCharges []model.ExtraCharges
	Debt         model.Debts
	DebtMonthStr string
	DownloadKeys string
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
