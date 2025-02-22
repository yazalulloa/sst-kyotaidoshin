package rates

import (
	"db/gen/model"
	"github.com/google/uuid"
	"kyotaidoshin/api"
	"log"
	"slices"
	"sync"
	"time"
)

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

			results[i] = Item{
				Key:        *api.Encode(*item.ID),
				CardId:     "rates-" + uuid.NewString(),
				Item:       item,
				DateOfRate: item.DateOfRate.Format(time.DateOnly),
				DateOfFile: item.DateOfFile.UnixMilli(),
				CreatedAt:  item.CreatedAt.UnixMilli(),
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

func CheckRateInsert(ratesArr *[]model.Rates) ([]model.Rates, error) {
	var wg sync.WaitGroup
	var once sync.Once
	var err error
	handleErr := func(e error) {
		if e != nil {
			once.Do(func() {
				err = e
			})
		}
	}

	wg.Add(len(*ratesArr))

	ratesToInsert := make([]model.Rates, len(*ratesArr))
	for i, rateToCheck := range *ratesArr {
		go func(rate model.Rates) {
			defer wg.Done()
			exists, err := CheckRateExist(*rate.ID)
			if err != nil {
				log.Printf("Error checking rate: %v", err)
				handleErr(err)
				return
			}
			if !exists {
				ratesToInsert[i] = rate
			}
		}(rateToCheck)
	}

	wg.Wait()

	if err != nil {
		return nil, err
	}

	ratesToInsert = slices.DeleteFunc(ratesToInsert, func(rate model.Rates) bool {
		return rate.FromCurrency == ""
	})

	return ratesToInsert, nil
}

func deleteRateReturnCounters(id int64, rateQuery RequestQuery) (*Counters, error) {

	_, err := deleteRateById(id)
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
		queryCount, err := getQueryCount(rateQuery)
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
