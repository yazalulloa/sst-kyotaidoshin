package apartments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"strings"
	"sync"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {
	var tableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		array, err := service.repo.selectList(requestQuery)
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
		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
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

func (service Service) deleteAndReturnCounters(keys Keys) (*Counters, error) {
	counters := Counters{}
	var rowsDeleted int64 = 0

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		rowsAffected, err := service.repo.deleteByKeys(keys)
		if err != nil {
			errorChan <- err
			return
		}

		rowsDeleted = rowsAffected
	}()

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()

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

func (service Service) Backup() (string, error) {

	requestQuery := RequestQuery{
		Limit: 30,
	}

	selectListDtos := func() ([]ApartmentDto, error) {
		list, err := service.repo.selectList(requestQuery)
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

func (service Service) ProcessDecoder(decoder *json.Decoder) (int64, error) {
	var dto []ApartmentDto
	err := decoder.Decode(&dto)
	if err != nil {
		log.Printf("Error decoding json: %s", err)
		return 0, err
	}

	array := make([]model.Apartments, len(dto))

	for i, apt := range dto {
		emails := strings.Join(apt.Emails, ",")
		array[i] = model.Apartments{
			BuildingID: apt.BuildingID,
			Number:     apt.Number,
			Name:       apt.Name,
			Aliquot:    apt.Aliquot,
			Emails:     emails,
		}
	}

	return service.repo.insertBulk(array)
}
