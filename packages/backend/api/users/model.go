package users

import (
	"db/gen/model"
	"github.com/google/uuid"
	"kyotaidoshin/util"
)

type Provider string

const (
	PLATFORM Provider = "PLATFORM"
	GOOGLE   Provider = "GOOGLE"
	GITHUB   Provider = "GITHUB"
)

func (receiver Provider) Name() string {
	return string(receiver)
}

type RequestQuery struct {
	LastId    string
	Limit     int
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
	CardId    string
	Key       string
	Item      model.Users
	CreatedAt int64
}

func cardId() string {
	return "users-" + uuid.NewString()
}
