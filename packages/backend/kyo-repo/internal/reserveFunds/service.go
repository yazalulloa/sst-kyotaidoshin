package reserveFunds

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) GetFormDto(buildingId string, receiptId string) (*FormDto, error) {

	reserveFunds, err := service.repo.SelectByBuilding(buildingId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(reserveFunds))

	for i, item := range reserveFunds {

		obj, err := toItem(&item, receiptId, nil)

		if err != nil {
			return nil, err
		}

		items[i] = *obj
	}

	return &FormDto{
		Key:   *util.Encode(Keys{BuildingId: buildingId}),
		Items: items,
	}, nil
}

func (service Service) getItem(id int32, receiptId string, oldCardId *string) (*Item, error) {
	item, err := service.repo.selectById(id)

	if err != nil {
		return nil, err
	}

	return toItem(item, receiptId, oldCardId)
}

func toItem(item *model.ReserveFunds, receiptId string, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, receiptId, cardIdStr)
	key := *util.Encode(keys)

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
