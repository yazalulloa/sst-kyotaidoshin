package apartments

import (
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
)

type Apt struct {
	Number string `json:"number"`
	Name   string `json:"name"`
}

type RequestQuery struct {
	lastBuildingId string
	lastNumber     string
	q              string
	buildings      []string
	Limit          int
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
	Item         model.Apartments
	Emails       []string
	UpdateParams *string
	isUpdate     bool
}

func cardId() string {
	return "apartments-" + uuid.NewString()
}

type Keys struct {
	BuildingId string
	Number     string
	CardId     string
}

func keys(apartments model.Apartments, cardId string) Keys {
	return Keys{
		BuildingId: apartments.BuildingID,
		Number:     apartments.Number,
		CardId:     cardId,
	}
}

type ApartmentDto struct {
	BuildingID string   `json:"building_id"`
	Number     string   `json:"number"`
	Name       string   `json:"name"`
	Aliquot    float64  `json:"aliquot"`
	Emails     []string `json:"emails"`
}

type UpdateParams struct {
	Key      string  `json:"key"`
	Building string  `json:"building"`
	Number   string  `json:"number"`
	Name     string  `json:"name"`
	IDDoc    string  `json:"id_doc"`
	Aliquot  float64 `json:"aliquot"`
	Emails   string  `json:"emails"`
}

type FormRequest struct {
	Key      string   `form:"key"`
	Building string   `form:"building" validate:"required_if=key ''"`
	Number   string   `form:"number" validate:"required_if=key ''"`
	Name     string   `form:"name" validate:"required,notblank,max=100"`
	Aliquot  float64  `form:"aliquot" validate:"required,gt=0"`
	Emails   []string `form:"emails" validate:"dive,email"`
}

type FormResponse struct {
	errorStr string
	item     *Item
}
