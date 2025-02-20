package extraCharges

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"kyotaidoshin/api"
	"strings"
)

func GetBuildingFormDto(buildingId string) (*FormDto, error) {

	extraCharges, err := selectByBuilding(buildingId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(extraCharges))

	for i, item := range extraCharges {

		obj, err := toItem(&item, nil)

		if err != nil {
			return nil, err
		}

		items[i] = *obj
	}

	return &FormDto{
		Key: *api.Encode(Keys{
			BuildingID:      buildingId,
			ParentReference: buildingId,
			Type:            TypeBuilding,
		}),
		Items: items,
	}, nil
}

func getItem(id int32, oldCardId *string) (*Item, error) {
	item, err := selectById(id)

	if err != nil {
		return nil, err
	}

	return toItem(item, oldCardId)
}

func toItem(item *model.ExtraCharges, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *api.Encode(keys)

	apts := strings.Split(item.Apartments, ",")

	updateParams := UpdateParams{
		Key:         key,
		Description: item.Description,
		Amount:      item.Amount,
		Currency:    item.Currency,
		Active:      item.Active,
		Apts:        apts,
	}

	byteArray, err := json.Marshal(updateParams)

	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	return &Item{
		CardId:       keys.CardId,
		Key:          key,
		Item:         *item,
		Apts:         apts,
		CreatedAt:    item.CreatedAt.UnixMilli(),
		UpdateParams: &base64Str,
	}, nil
}
