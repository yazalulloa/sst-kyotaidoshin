package reserveFunds

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"kyotaidoshin/api"
)

func GetFormDto(buildingId string) (*FormDto, error) {

	reserveFunds, err := SelectByBuilding(buildingId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(reserveFunds))

	for i, item := range reserveFunds {

		obj, err := toItem(&item, nil)

		if err != nil {
			return nil, err
		}

		items[i] = *obj
	}

	return &FormDto{
		Key:   *api.Encode(Keys{BuildingId: buildingId}),
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

func toItem(item *model.ReserveFunds, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *api.Encode(keys)

	updateParams := UpdateParams{
		Key:           key,
		Name:          item.Name,
		Fund:          item.Fund,
		Expense:       item.Expense,
		Pay:           item.Pay,
		Active:        item.Active,
		Type:          item.Type,
		ExpenseType:   item.ExpenseType,
		AddToExpenses: item.AddToExpenses,
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
		CreatedAt:    item.CreatedAt.UnixMilli(),
		UpdateParams: &base64Str,
	}, nil
}
