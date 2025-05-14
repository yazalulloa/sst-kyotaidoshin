package rates

import (
	"context"
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"sync"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func ToItem(rate model.Rates) Item {
	return Item{
		Key:        *util.Encode(*rate.ID),
		CardId:     "rates-" + uuid.NewString(),
		Item:       rate,
		DateOfRate: rate.DateOfRate.Format(time.DateOnly),
		DateOfFile: rate.DateOfFile.UnixMilli(),
		CreatedAt:  rate.CreatedAt.UnixMilli(),
	}
}

func (service Service) getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var tableResponse TableResponse
	var oErr error
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		array, err := service.repo.SelectList(requestQuery)
		results := make([]Item, len(array))
		for i, item := range array {
			results[i] = ToItem(item)
		}
		tableResponse.Results = results
		oErr = err
	}()

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()
		tableResponse.Counters.TotalCount = totalCount
		oErr = err
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}

		oErr = err
	}()

	wg.Wait()
	return tableResponse, oErr
}

func (service Service) deleteRateReturnCounters(id int64, requestQuery RequestQuery) (*Counters, error) {

	_, err := service.repo.deleteRateById(id)
	if err != nil {
		return nil, err
	}

	var counters Counters
	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}
		counters.QueryCount = queryCount
	}()

	wg.Wait()
	close(errorChan)

	if err := util.HasErrors(errorChan); err != nil {
		return nil, err
	}

	return &counters, nil

}
