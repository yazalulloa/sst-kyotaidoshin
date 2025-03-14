package extraCharges

import (
	"db/gen/model"
	"github.com/google/uuid"
)

const TypeBuilding string = "BUILDING"
const TypeReceipt string = "RECEIPT"

type Item struct {
	CardId       string
	Key          string
	Item         model.ExtraCharges
	Apts         []string
	CreatedAt    int64
	UpdateParams *string
	isUpdate     *bool
}

func cardId() string {
	return "extra-charges-" + uuid.NewString()
}

type Keys struct {
	Id              *int32
	BuildingID      string
	ParentReference string
	Type            string
	CardId          string
}

func keys(item model.ExtraCharges, cardId string) Keys {
	return Keys{
		Id:              item.ID,
		BuildingID:      item.BuildingID,
		ParentReference: item.ParentReference,
		Type:            item.Type,
		CardId:          cardId,
	}
}

type FormDto struct {
	Key   string
	Items []Item
}

type UpdateParams struct {
	Key         string   `json:"key"`
	Description string   `json:"description"`
	Amount      float64  `json:"amount"`
	Currency    string   `json:"currency"`
	Active      bool     `json:"active"`
	Apts        []string `json:"apts"`
}
type FormResponse struct {
	errorStr string
	item     *Item
	counter  int64
}

type FormRequest struct {
	Key         string   `form:"key" validate:"required,min=3,max=500"`
	Description string   `form:"description" validate:"required,min=3,max=100"`
	Amount      float64  `form:"amount" validate:"required,gt=0"`
	Currency    string   `form:"currency" validate:"required,oneof=USD VED"`
	Active      bool     `form:"active"`
	Apartments  []string `form:"apartment_input" validate:"required,gt=0,dive,notblank"`
}
type AptDto struct {
	Number string `json:"number"`
}
type ExtraChargeDto struct {
	BuildingID      string   `json:"building_id"`
	ParentReference string   `json:"parent_reference"`
	Type            string   `json:"type"`
	Description     string   `json:"description"`
	Amount          float64  `json:"amount"`
	Currency        string   `json:"currency"`
	Active          bool     `json:"active"`
	Apartments      []AptDto `json:"apartments"`
}
