package reserveFunds

import "kyotaidoshin/api"

func GetFormDto(buildingId string) (*FormDto, error) {

	reserveFunds, err := SelectByBuilding(buildingId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(reserveFunds))

	for i, item := range reserveFunds {
		items[i] = Item{
			CardId:    cardId(),
			Key:       *api.Encode(keys(item)),
			Item:      item,
			CreatedAt: item.CreatedAt.UnixMilli(),
		}
	}

	return &FormDto{
		Key:   *api.Encode(Keys{BuildingId: buildingId}),
		Items: items,
	}, nil
}
