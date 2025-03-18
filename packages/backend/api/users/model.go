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
	CardId       string
	Key          string
	Item         model.Users
	Role         *model.Roles
	CreatedAt    int64
	LastLoginAt  int64
	isUpdate     bool
	UpdateParams *string
}

func cardId() string {
	return "users-" + uuid.NewString()
}

type Keys struct {
	ID     string
	CardId string
}

func keys(item model.Users, cardId string) Keys {
	return Keys{
		ID:     item.ID,
		CardId: cardId,
	}
}

type UpdateParams struct {
	Key    string `json:"key"`
	RoleId *int32 `json:"role_id"`

	Provider string `json:"provider"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Picture  string `json:"picture"`
}

type FormRequest struct {
	Key    string `form:"key"`
	RoleId int32  `form:"role"`
}

type FormResponse struct {
	errorStr string
	item     *Item
}
