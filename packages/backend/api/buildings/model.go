package buildings

import (
	"db/gen/model"
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

type FormDto struct {
	isEdit                      bool
	key                         *string
	emailConfigs                []EmailConfig
	building                    *model.Buildings
	reserveFundFormDto          reserveFunds.FormDto
	extraChargesFormDto         extraCharges.FormDto
	apts                        string
	currencies                  string
	currenciesToShowAmountToPay string
}

type EmailConfig struct {
	id    string
	key   string
	email string
}

type FormResponse struct {
	createdNew *bool
	key        *string
	errorStr   string
}
