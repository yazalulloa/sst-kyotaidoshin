package debts

import (
	"encoding/base64"
	"encoding/json"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
)

func GetFormDto(buildingId string, receiptId string) (*FormDto, error) {

	list, err := SelectByBuildingReceipt(buildingId, receiptId)

	if err != nil {
		return nil, err
	}

	items := make([]Item, len(list))

	totals := Totals{
		Counter: len(list),
	}

	for i, item := range list {
		totals.TotalReceipts += item.Receipts
		totals.TotalAmount += item.Amount

		obj, err := toItem(&item, nil)

		if err != nil {
			return nil, err
		}

		items[i] = *obj
	}

	return &FormDto{
		Items:  items,
		Totals: totals,
	}, nil
}

func toItem(item *model.Debts, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *util.Encode(keys)

	var months MonthlyDebt

	if item.Months != "" {

		err := json.Unmarshal([]byte(item.Months), &months)
		if err != nil {
			return nil, err
		}
	}

	updateParams := UpdateParams{
		Key:                           key,
		Apt:                           item.AptNumber,
		Receipts:                      item.Receipts,
		Amount:                        item.Amount,
		Months:                        months,
		PreviousPaymentAmount:         item.PreviousPaymentAmount,
		PreviousPaymentAmountCurrency: item.PreviousPaymentAmountCurrency,
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
		UpdateParams: &base64Str,
		Months:       months,
	}, nil
}
