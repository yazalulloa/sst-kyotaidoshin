package rates

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
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

func (service Service) UpdateStableTrend() error {

	list, err := service.repo.byTrend(STABLE)
	if err != nil {
		return fmt.Errorf("error fetching stable trend rates: %w", err)
	}

	length := len(list)
	if length == 0 {
		log.Println("no stable trend rates found")
		return nil
	}

	log.Printf("found %d stable trend rates", length)

	var wg sync.WaitGroup
	wg.Add(length)
	rateChan := make(chan *model.Rates, length)
	errorChan := make(chan error, length)

	for _, rate := range list {

		go func() {
			defer wg.Done()
			req := RequestQuery{
				LastId:     *rate.ID,
				Currencies: []string{rate.FromCurrency},
				Limit:      1,
				SortOrder:  util.SortOrderTypeDESC,
			}
			previousList, err := service.repo.SelectList(req)
			if err != nil {
				errorChan <- fmt.Errorf("error fetching previous rate for ID %d %s: %w", *rate.ID, rate.FromCurrency, err)
				return
			}

			if len(previousList) == 0 {
				log.Printf("no previous rate found for ID %d %s", *rate.ID, rate.FromCurrency)
				return
			}

			previousRate := previousList[0]
			rateToModified := &rate

			CalculateTrend(rateToModified, &previousRate)

			rateChan <- rateToModified

		}()
	}

	wg.Wait()
	close(rateChan)
	close(errorChan)

	if err := util.HasErrors(errorChan); err != nil {
		return fmt.Errorf("error updating stable trend rates: %w", err)
	}

	var ratesToUpdate []*model.Rates
	for rate := range rateChan {
		ratesToUpdate = append(ratesToUpdate, rate)
	}

	if len(ratesToUpdate) == 0 {
		log.Println("no stable trend rates to update")
		return nil
	}

	rowsAffected, err := service.repo.Insert(ratesToUpdate)
	if err != nil {
		return fmt.Errorf("error updating stable trend rates: %w", err)
	}

	log.Printf("updated %d stable trend rates %d", rowsAffected, len(ratesToUpdate))

	return nil
}

func CalculateTrend(lhs, rhs *model.Rates) {
	diff := 0.00
	diffPercent := 0.00
	trend := STABLE
	if rhs != nil {
		previousRate := lhs.Rate
		nextRate := rhs.Rate

		diff = previousRate - nextRate

		if nextRate != 0 {
			diffPercent = (math.Abs(diff) / nextRate) * 100
			diffPercent = util.RoundFloat(diffPercent, 2)
		}

		if previousRate > nextRate {
			trend = UP
		} else if previousRate < nextRate {
			trend = DOWN
		}
	}

	lhs.ToCurrency = "VED"
	lhs.Source = "BCV"
	lhs.Trend = trend.Name()
	lhs.Diff = diff
	lhs.DiffPercent = diffPercent
}
