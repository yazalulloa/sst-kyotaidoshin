package rates

import (
	"context"
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"sync"
	"time"
)

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

func getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var tableResponse TableResponse
	var oErr error
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		array, err := SelectList(requestQuery)
		results := make([]Item, len(array))
		for i, item := range array {
			results[i] = ToItem(item)
		}
		tableResponse.Results = results
		oErr = err
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		tableResponse.Counters.TotalCount = totalCount
		oErr = err
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(requestQuery)
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}

		oErr = err
	}()

	wg.Wait()
	return tableResponse, oErr
}

func deleteRateReturnCounters(ctx context.Context, id int64, requestQuery RequestQuery) (*Counters, error) {

	_, err := deleteRateById(id)
	if err != nil {
		return nil, err
	}

	var counters Counters
	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(requestQuery)
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
