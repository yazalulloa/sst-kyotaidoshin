package buildings

import (
	"db/gen/model"
	"kyotaidoshin/apartments"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"time"
)

type RequestQuery struct {
	LastCreatedAt *time.Time
	Limit         int
	SortOrder     util.SortOrderType
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
	CardId    string
	Key       string
	Item      model.Buildings
	AptCount  int64
	CreatedAt int64
}

type UpdateParams struct {
	ID                          string           `json:"id"`
	Name                        string           `json:"name"`
	Rif                         string           `json:"rif"`
	MainCurrency                string           `json:"mainCurrency"`
	DebtCurrency                string           `json:"debtCurrency"`
	CurrenciesToShowAmountToPay []string         `json:"currenciesToShowAmountToPay"`
	DebtsCurrenciesToShow       []string         `json:"debtsCurrenciesToShow"`
	FixedPay                    bool             `json:"fixedPay"`
	FixedPayAmount              float64          `json:"fixedPayAmount"`
	RoundUpPayments             bool             `json:"roundUpPayments"`
	EmailConfig                 string           `json:"emailConfig"`
	Apts                        []apartments.Apt `json:"apts"`
}

type FormDto struct {
	emailConfigs        []EmailConfig
	reserveFundFormDto  reserveFunds.FormDto
	extraChargesFormDto extraCharges.FormDto
	UpdateParams        *string
	Key                 *string
}

type EmailConfig struct {
	key   string
	email string
}

type FormResponse struct {
	createdNew *bool
	key        *string
	errorStr   string
}

const idMinLen = 3
const idMaxLen = 20
const nameMinLen = 3
const nameMaxLen = 100
const rifMinLen = 7
const rifMaxLen = 20
const currencyMaxLen = 3
const fixedPayAmountMaxLen = 18

type FormRequest struct {
	Key                         *string  `form:"key"`
	Id                          string   `form:"id" validate:"required_if=Key nil,min=3,max=20,alphanumunicode"`
	Name                        string   `form:"name" validate:"required,min=3,max=100"`
	Rif                         string   `form:"rif" validate:"required,min=7,max=20"`
	MainCurrency                string   `form:"mainCurrency" validate:"required,oneof=USD VED"`
	DebtCurrency                string   `form:"debtCurrency" validate:"required,oneof=USD VED"`
	CurrenciesToShowAmountToPay []string `form:"currenciesToShowAmountToPay" validate:"dive,oneof=USD VED"`
	DebtsCurrenciesToShow       []string `form:"debtsCurrenciesToShow" validate:"dive,oneof=USD VED"`
	RoundUpPayments             bool     `form:"roundUpPayments"`
	FixedPay                    bool     `form:"fixedPay"`
	FixedPayAmount              float64  `form:"fixedPayAmount" validate:"required_if=fixedPay true"`
	EmailConfig                 string   `form:"emailConfig" validate:"required"`
}
