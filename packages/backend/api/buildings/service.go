package buildings

import (
	"github.com/google/uuid"
	"kyotaidoshin/extraCharges"
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

	var wg sync.WaitGroup
	workers := 3
	wg.Add(workers)
	errorChan := make(chan error, workers)

	go func() {
		defer wg.Done()
		_, err := deleteById(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := reserveFunds.DeleteByBuilding(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := extraCharges.DeleteByBuilding(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	totalCount, err := getTotalCount()
	if err != nil {
		return nil, err
	}

	counters := Counters{}
	counters.TotalCount = totalCount
	return &counters, nil
}
