package apartments

import (
	"github.com/google/uuid"
	"kyotaidoshin/api"
	"sync"
)

func getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var tableResponse TableResponse
	var oErr error
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		array, err := selectList(requestQuery)
		results := make([]Item, len(array))
		for i, item := range array {

			results[i] = Item{
				Key:    *api.Encode(keys(item)),
				CardId: "apartments-" + uuid.NewString(),
				Item:   item,
			}

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

func deleteAndReturnCounters(keys Keys) (*Counters, error) {
	counters := Counters{}
	var rowsDeleted int64 = 0
	var oErr error
	var once sync.Once
	handleErr := func(e error) {
		if e != nil {
			once.Do(func() {
				oErr = e
			})
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rowsAffected, err := deleteByKeys(keys)
		handleErr(err)
		rowsDeleted = rowsAffected
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		counters.TotalCount = totalCount
		handleErr(err)
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	counters.TotalCount -= rowsDeleted
	return &counters, nil
}
