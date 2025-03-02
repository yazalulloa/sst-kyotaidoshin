package buildings

import (
	"github.com/google/uuid"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"sync"
)

func getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var rateTableResponse TableResponse
	var oErr error
	var wg sync.WaitGroup
	var once sync.Once
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
		array, err := selectList(requestQuery)
		if err != nil {
			handleErr(err)
			return
		}

		results := make([]Item, len(array))
		for i, item := range array {
			//log.Printf("ID %d aptCount %d\n", item.ID, *item.AptCount)

			results[i] = Item{
				Key:       *util.Encode(item.ID),
				CardId:    "buildings-" + uuid.NewString(),
				Item:      item.Buildings,
				AptCount:  item.AptCount,
				CreatedAt: item.CreatedAt.UnixMilli(),
			}

		}
		rateTableResponse.Results = results
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		if err != nil {
			handleErr(err)
			return
		}
		rateTableResponse.Counters.TotalCount = totalCount
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
	wg.Add(3)

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

	go func() {
		defer wg.Done()
		_, err := reserveFunds.DeleteByBuilding(id)
		handleErr(err)
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	counters.TotalCount -= rowsDeleted
	return &counters, nil
}
