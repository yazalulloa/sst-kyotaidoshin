package apartments

import (
	"encoding/base64"
	"encoding/json"
	"kyo-repo/internal/api"
	"kyo-repo/internal/db/gen/model"
	"kyo-repo/internal/util"
	"strings"
	"sync"
)

func getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {
	var tableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		array, err := selectList(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}

		items := make([]Item, len(array))
		for i, item := range array {

			obj, err := toItem(&item, nil)
			if err != nil {
				errorChan <- err
				return
			}

			items[i] = *obj

		}
		tableResponse.Results = items
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	return &tableResponse, nil
}

func toItem(item *model.Apartments, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *util.Encode(keys)

	updateParams := UpdateParams{
		Key:      key,
		Building: item.BuildingID,
		Number:   item.Number,
		Name:     item.Name,
		IDDoc:    item.IDDoc,
		Aliquot:  item.Aliquot,
		Emails:   item.Emails,
	}

	byteArray, err := json.Marshal(updateParams)

	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	emails := strings.Split(item.Emails, ",")

	return &Item{
		CardId:       keys.CardId,
		Key:          key,
		Item:         *item,
		Emails:       emails,
		UpdateParams: &base64Str,
	}, nil
}

func deleteAndReturnCounters(keys Keys) (*Counters, error) {
	counters := Counters{}
	var rowsDeleted int64 = 0

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		rowsAffected, err := deleteByKeys(keys)
		if err != nil {
			errorChan <- err
			return
		}

		rowsDeleted = rowsAffected
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()

		if err != nil {
			errorChan <- err
			return
		}

		counters.TotalCount = totalCount
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	counters.TotalCount -= rowsDeleted
	return &counters, nil
}

func insertDtos(apts []ApartmentDto) (int64, error) {

	array := make([]model.Apartments, len(apts))

	for i, apt := range apts {
		emails := strings.Join(apt.Emails, ",")
		array[i] = model.Apartments{
			BuildingID: apt.BuildingID,
			Number:     apt.Number,
			Name:       apt.Name,
			Aliquot:    apt.Aliquot,
			Emails:     emails,
		}
	}

	return insertBulk(array)
}

func Backup() (string, error) {

	requestQuery := RequestQuery{
		Limit: 30,
	}

	selectListDtos := func() ([]ApartmentDto, error) {
		list, err := selectList(requestQuery)
		if err != nil {
			return nil, err
		}

		dtos := make([]ApartmentDto, len(list))

		for i, item := range list {
			dtos[i] = ApartmentDto{
				BuildingID: item.BuildingID,
				Number:     item.Number,
				Name:       item.Name,
				Aliquot:    item.Aliquot,
				Emails:     strings.Split(item.Emails, ","),
			}

			if i == len(list)-1 {
				requestQuery.lastBuildingId = item.BuildingID
				requestQuery.lastNumber = item.Number
			}
		}

		return dtos, nil
	}

	return api.Backup(api.BACKUP_APARTMENTS_FILE, selectListDtos)
}
