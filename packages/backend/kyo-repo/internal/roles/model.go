package roles

import (
	"github.com/google/uuid"
	"kyo-repo/internal/db/gen/model"
	"kyo-repo/internal/util"
)

type RequestQuery struct {
	LastId    int32
	Q         string
	Limit     int
	SortOrder util.SortOrderType
}
type Item struct {
	CardId       string
	Key          string
	Item         RoleWithPermissions
	isUpdate     bool
	UpdateParams *string
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

func cardId() string {
	return "roles-" + uuid.NewString()
}

type Keys struct {
	ID     int32
	CardId string
}

func keys(item model.Roles, cardId string) Keys {
	return Keys{
		ID:     *item.ID,
		CardId: cardId,
	}
}

type FormRequest struct {
	Key   string  `form:"key"`
	Name  string  `form:"name" validate:"required,notblank,max=100"`
	Perms []int32 `form:"perms" validate:"dive,required,gt=0"`
}

type FormResponse struct {
	errorStr string
	item     *Item
}

type RoleWithPermissions struct {
	Role        model.Roles
	Permissions []model.Permissions
}

type UpdateParams struct {
	Key   string  `json:"key"`
	Name  string  `json:"name"`
	Perms []int32 `json:"perms"`
}

type RoleMin struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	PermsLength int    `json:"perms"`
}
