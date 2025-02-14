package buildings

import (
	"github.com/google/uuid"
	"kyotaidoshin/api"
	"sync"
)

func getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var rateTableResponse TableResponse
	var oErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		array, err := selectList(requestQuery)
		results := make([]Item, len(array))
		for i, item := range array {

			results[i] = Item{
				Key:       *api.Encode(item.ID),
				CardId:    "buildings-" + uuid.NewString(),
				Item:      item,
				CreatedAt: item.CreatedAt.UnixMilli(),
			}

		}
		rateTableResponse.Results = results
		oErr = err
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		rateTableResponse.Counters.TotalCount = totalCount
		oErr = err
	}()

	wg.Wait()
	return rateTableResponse, oErr
}

func deleteAndReturnCounters(id string) (*Counters, error) {
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
		rowsAffected, err := deleteById(id)
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
