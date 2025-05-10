package users

import (
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
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

type EventNotifications string

const (
	NEW_RATE EventNotifications = "new_rate"
	NEW_USER EventNotifications = "new_user"
)

func (receiver EventNotifications) Name() string {
	return string(receiver)
}

func (reciever EventNotifications) is(str string) bool {
	return reciever.Name() == str
}

func isEventNotifications(str string) bool {
	for _, v := range []EventNotifications{NEW_RATE, NEW_USER} {
		if v.is(str) {
			return true
		}
	}
	return false
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
	CardId             string
	Key                string
	Item               model.Users
	Role               *model.Roles
	Chat               *model.TelegramChats
	CreatedAt          int64
	LastLoginAt        int64
	NotificationEvents []string
	isUpdate           bool
	UpdateParams       *string
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
	Key                string   `json:"key"`
	RoleId             *int32   `json:"role_id"`
	NotificationEvents []string `json:"notification_events"`

	Provider string `json:"provider"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Picture  string `json:"picture"`
}

type FormRequest struct {
	Key                string   `form:"key"`
	RoleId             int32    `form:"role"`
	NotificationEvents []string `form:"notification_events"`
}

type FormResponse struct {
	errorStr string
	item     *Item
}
