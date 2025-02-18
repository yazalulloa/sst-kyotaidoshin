package reserveFunds

import (
	"db/gen/model"
	"github.com/google/uuid"
)

type Item struct {
	CardId    string
	Key       string
	Item      model.ReserveFunds
	CreatedAt int64
}

func cardId() string {
	return "reserve-funds-" + uuid.NewString()
}

type Keys struct {
	BuildingId string
	Id         int32
	ReceiptId  *int64
	CardId     string
}

func keys(item model.ReserveFunds) Keys {
	return Keys{
		BuildingId: item.BuildingID,
		Id:         *item.ID,
		ReceiptId:  nil,
		CardId:     cardId(),
	}
}

type FormDto struct {
	Key   string
	Items []Item
}
