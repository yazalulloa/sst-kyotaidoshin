package users

import (
	"kyotaidoshin/util"
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
				Key:       *util.Encode(item.ID),
				CardId:    cardId(),
				Item:      item,
				CreatedAt: item.CreatedAt.UnixMilli(),
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

func deleteRateReturnCounters(id string, requestQuery RequestQuery) (*Counters, error) {

	_, err := deleteById(id)
	if err != nil {
		return nil, err
	}

	var counters Counters
	var wg sync.WaitGroup
	var once sync.Once
	var oErr error
	handleErr := func(e error) {
		if e != nil {
			once.Do(func() {
				oErr = e
			})
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		if err != nil {
			handleErr(err)
			return
		}
		counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(requestQuery)
		if err != nil {
			handleErr(err)
			return
		}
		counters.QueryCount = queryCount
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	return &counters, nil

}
