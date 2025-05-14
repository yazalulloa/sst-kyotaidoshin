package extraCharges

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) GetBuildingFormDto(buildingId string) (*FormDto, error) {

	list, err := service.repo.SelectByBuilding(buildingId)

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
		Key: *util.Encode(Keys{
			BuildingID:      buildingId,
			ParentReference: buildingId,
			Type:            TypeBuilding,
		}),
		Items: items,
	}, nil
}

func (service Service) GetReceiptFormDto(buildingId string, receiptId string) (*FormDto, error) {

	list, err := service.repo.SelectByReceipt(receiptId)

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
		Key: *util.Encode(Keys{
			BuildingID:      buildingId,
			ParentReference: fmt.Sprint(receiptId),
			Type:            TypeReceipt,
		}),
		Items: items,
	}, nil
}

func (service Service) getItem(id int32, oldCardId *string) (*Item, error) {
	item, err := service.repo.selectById(id)

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
	key := *util.Encode(keys)

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
