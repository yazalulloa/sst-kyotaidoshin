package expenses

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"kyotaidoshin/api"
)

func GetFormDto(buildingId string, receiptId int32) (*FormDto, error) {

	list, err := SelectByReceipt(receiptId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(list))

	for i, item := range list {

		obj, err := toItem(&item, nil)

		if err != nil {
			return nil, err
		}

		items[i] = *obj
	}

	return &FormDto{
		Key: *api.Encode(Keys{
			BuildingID: buildingId,
			ReceiptID:  receiptId,
		}),
		Items: items,
	}, nil
}

func toItem(item *model.Expenses, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, &cardIdStr)
	key := *api.Encode(keys)

	updateParams := UpdateParams{
		Key:         key,
		Description: item.Description,
		Amount:      item.Amount,
		Currency:    item.Currency,
		Type:        item.Type,
	}

	byteArray, err := json.Marshal(updateParams)

	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	return &Item{
		CardId:       *keys.CardId,
		Key:          key,
		Item:         *item,
		UpdateParams: &base64Str,
	}, nil
}
