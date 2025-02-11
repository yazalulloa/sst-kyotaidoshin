package buildings

import (
	"db/gen/model"
	"kyotaidoshin/apartments"
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
	CreatedAt int64
}

type FormDto struct {
	isEdit                      bool
	key                         *string
	emailConfigs                []EmailConfig
	building                    *model.Buildings
	apts                        []apartments.Apt
	currencies                  string
	currenciesToShowAmountToPay string
}

type EmailConfig struct {
	id    string
	key   string
	email string
}
