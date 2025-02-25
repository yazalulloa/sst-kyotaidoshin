package receipts

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"kyotaidoshin/apartments"
	"kyotaidoshin/api"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

func loadRates(records []ReceiptRecord, ratesHolder *RatesHolder) error {
	for _, record := range records {
		if record.Receipt.Date == "" {
			log.Printf("Record has no date: %v", record)
			return errors.New("Record has no date")
		}

		value, ok := ratesHolder.Rates.Load(record.Receipt.Date)
		if !ok || value == 0 {
			ratesHolder.Rates.Store(record.Receipt.Date, 0)
		}
	}

	ratesToLookUp := make([]string, 0)
	ratesHolder.Rates.Range(func(key, value interface{}) bool {
		if value == 0 {
			ratesToLookUp = append(ratesToLookUp, key.(string))
		}
		return true
	})

	if len(ratesToLookUp) == 0 {
		log.Printf("No rates to look up")
		return nil
	}

	log.Printf("Looking up rates for dates: %d", len(ratesToLookUp))

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

	wg.Add(len(ratesToLookUp))

	for _, date := range ratesToLookUp {

		go func(date string) {
			defer wg.Done()

			parsedDate, err := time.Parse(time.DateOnly, date)
			if err != nil {
				handleErr(err)
				return
			}

			rate, err := rates.GetFirstBeforeDate("USD", parsedDate)
			if err != nil {
				handleErr(err)
				return
			}

			ratesHolder.Rates.Store(date, *rate.ID)
		}(date)
	}

	wg.Wait()

	if oErr != nil {
		log.Printf("Error getting rates: %s", oErr)
		return oErr
	}

	return nil
}

func insertRecord(records []ReceiptRecord, ratesHolder *RatesHolder) (int64, error) {

	err := loadRates(records, ratesHolder)
	if err != nil {
		return 0, err
	}

	var extraChargeArray []model.ExtraCharges
	var expensesArray []model.Expenses
	var debtsArray []model.Debts

	var counter int64
	for _, record := range records {

		date, err := time.Parse(time.DateOnly, record.Receipt.Date)
		if err != nil {
			return 0, err
		}
		var lastSent *time.Time

		if record.Receipt.LastSent != nil && *record.Receipt.LastSent != "" {
			tmp, err := time.Parse(time.RFC3339, *record.Receipt.LastSent+"Z")
			if err != nil {
				return 0, err
			}
			lastSent = &tmp
		}

		rateId, _ := ratesHolder.Rates.Load(record.Receipt.Date)

		createdAt := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC)

		receipt := model.Receipts{
			BuildingID: record.Receipt.BuildingID,
			Year:       record.Receipt.Year,
			Month:      record.Receipt.Month,
			Date:       date,
			RateID:     rateId.(int64),
			Sent:       record.Receipt.Sent,
			LastSent:   lastSent,
			CreatedAt:  &createdAt,
		}

		receiptId, err := insertBackup(receipt)
		if err != nil {
			return 0, err
		}
		counter++

		for _, extraCharge := range record.ExtraCharges {
			var builder strings.Builder
			for idx, apt := range extraCharge.Apartments {
				builder.WriteString(apt.Number)
				if idx < len(extraCharge.Apartments)-1 {
					builder.WriteString(",")
				}
			}

			parentReference := strconv.FormatInt(receiptId, 10)
			extraChargeArray = append(extraChargeArray, model.ExtraCharges{
				BuildingID:      extraCharge.BuildingID,
				ParentReference: parentReference,
				Type:            extraCharges.TypeReceipt,
				Description:     extraCharge.Description,
				Amount:          extraCharge.Amount,
				Currency:        extraCharge.Currency,
				Active:          extraCharge.Active,
				Apartments:      builder.String(),
			})
		}

		for _, expense := range record.Expenses {

			expensesArray = append(expensesArray, model.Expenses{
				BuildingID:  expense.BuildingID,
				ReceiptID:   int32(receiptId),
				Description: expense.Description,
				Amount:      expense.Amount,
				Currency:    expense.Currency,
				Type:        expense.Type,
			})

		}

		for _, debt := range record.Debts {
			stringArray := make([]string, len(debt.Months))

			for i, num := range debt.Months {
				stringArray[i] = strconv.Itoa(int(num))
			}
			months := strings.Join(stringArray, ",")

			debtsArray = append(debtsArray, model.Debts{
				BuildingID:                    debt.BuildingID,
				ReceiptID:                     int32(receiptId),
				AptNumber:                     debt.AptNumber,
				Receipts:                      debt.Receipts,
				Amount:                        debt.Amount,
				Months:                        months,
				PreviousPaymentAmount:         debt.PreviousPaymentAmount,
				PreviousPaymentAmountCurrency: debt.PreviousPaymentAmountCurrency,
			})
		}

	}

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

	wg.Add(3)

	go func() {
		defer wg.Done()
		rows, err := extraCharges.InsertBackup(extraChargeArray)
		handleErr(err)
		log.Printf("Inserted %d/%d extra charges", len(extraChargeArray), rows)
	}()

	go func() {
		defer wg.Done()
		rows, err := expenses.InsertBackup(expensesArray)
		handleErr(err)
		log.Printf("Inserted %d/%d expenses", len(expensesArray), rows)
	}()

	go func() {
		defer wg.Done()

		rows, err := debts.InsertBackup(debtsArray)
		handleErr(err)
		log.Printf("Inserted %d/%d debts", len(debtsArray), rows)
	}()

	wg.Wait()

	if oErr != nil {
		return 0, oErr
	}

	return counter, nil
}

func getTableResponse(requestQuery RequestQuery) (TableResponse, error) {
	var tableResponse TableResponse
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
		array, err := selectList(requestQuery)
		if err != nil {
			handleErr(err)
			return
		}

		items := make([]Item, len(array))
		for i, item := range array {

			obj, err := toItem(&item, nil)
			if err != nil {
				handleErr(err)
				return
			}

			items[i] = *obj

		}
		tableResponse.Results = items
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount()
		if err != nil {
			handleErr(err)
			return
		}
		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(requestQuery)
		if err != nil {
			handleErr(err)
			return
		}
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}
	}()

	wg.Wait()

	return tableResponse, oErr
}

func toItem(item *model.Receipts, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *api.Encode(keys)

	//updateParams := UpdateParams{
	//	Key:    key,
	//	Year:   item.Year,
	//	Month:  item.Month,
	//	Date:   item.Date.Format(time.DateOnly),
	//}

	//byteArray, err := json.Marshal(updateParams)
	//
	//if err != nil {
	//	return nil, err
	//}
	//
	//base64Str := base64.URLEncoding.EncodeToString(byteArray)

	var lastSent *int64
	if item.LastSent != nil {
		tmp := item.LastSent.UnixMilli()
		lastSent = &tmp
	}

	return &Item{
		CardId:    keys.CardId,
		Key:       key,
		Item:      *item,
		CreatedAt: item.CreatedAt.UnixMilli(),
		//UpdateParams: &base64Str,
		LastSent: lastSent,
	}, nil
}

func getFormDto(keys Keys) (*FormDto, error) {
	formDto := FormDto{}

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
	wg.Add(7)

	go func() {
		defer wg.Done()

		building, err := buildings.SelectById(keys.BuildingId)
		if err != nil {
			handleErr(err)
			return
		}

		formDto.building = *building
	}()

	go func() {
		defer wg.Done()

		receipt, err := selectById(keys.Id)
		if err != nil {
			handleErr(err)
			return
		}

		if receipt == nil {
			handleErr(errors.New("receipt not found"))
			return
		}

		ratesDtos, err := getRatesDtos(&receipt.Date)
		if err != nil {
			handleErr(err)
			return
		}

		newKeys := Keys{
			BuildingId: receipt.BuildingID,
			Id:         *receipt.ID,
		}

		newKeysStr := api.Encode(newKeys)

		updateParams := UpdateParams{
			Key:   *newKeysStr,
			Year:  receipt.Year,
			Month: receipt.Month,
			Date:  receipt.Date.Format(time.DateOnly),
		}

		byteArray, err := json.Marshal(updateParams)

		if err != nil {
			handleErr(err)
			return
		}

		base64Str := base64.URLEncoding.EncodeToString(byteArray)

		formDto.key = *newKeysStr
		formDto.receipt = receipt
		formDto.rates = ratesDtos
		formDto.updateParams = base64Str
	}()

	go func() {
		defer wg.Done()

		dto, err := expenses.GetFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			handleErr(err)
			return
		}

		formDto.expenseFormDto = *dto
	}()

	go func() {
		defer wg.Done()

		reserveFundFormDto, err := reserveFunds.GetFormDto(keys.BuildingId, &keys.Id)
		if err != nil {
			handleErr(err)
			return
		}

		formDto.reserveFundFormDto = *reserveFundFormDto
	}()

	go func() {
		defer wg.Done()

		dto, err := extraCharges.GetReceiptFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			handleErr(err)
			return
		}

		formDto.extraChargesFormDto = *dto
	}()

	go func() {
		defer wg.Done()

		dto, err := debts.GetFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			handleErr(err)
			return
		}

		formDto.debtFormDto = *dto
	}()

	go func() {
		defer wg.Done()
		apts, err := apartments.SelectNumberAndNameByBuildingId(keys.BuildingId)
		if err != nil {
			handleErr(err)
			return
		}

		aptStr, err := json.Marshal(apts)
		if err != nil {
			handleErr(err)
			return
		}

		base64Str := base64.URLEncoding.EncodeToString(aptStr)

		formDto.apts = base64Str
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	var rate float64 = 0
	for _, v := range formDto.rates {
		if v.ID == formDto.receipt.RateID {
			rate = v.Rate
			break
		}
	}

	if rate == 0 {
		rate = formDto.rates[0].Rate
	}

	receiptExpensesDto := GetReceiptExpensesDto(keys.Id, formDto.expenseFormDto.Items, formDto.reserveFundFormDto.Items)

	formDto.expenseTotals = receiptExpensesDto.Totals
	formDto.reserveFundExpenses = receiptExpensesDto.ReserveFundExpenses

	return &formDto, nil
}

func getRatesDtos(date *time.Time) ([]RateDto, error) {
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

	var firstRateArray []model.Rates
	var secondRateArray []model.Rates

	go func() {
		defer wg.Done()
		arr, err := rates.GetFromDate("USD", *date, 5, true)

		if err != nil {
			handleErr(err)
			return
		}

		slices.Reverse(arr)
		firstRateArray = arr
	}()

	go func() {
		defer wg.Done()
		arr, err := rates.GetFromDate("USD", *date, 5, false)

		if err != nil {
			handleErr(err)
			return
		}

		secondRateArray = arr
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	firstLen := len(firstRateArray)
	ratesDto := make([]RateDto, len(firstRateArray)+len(secondRateArray))

	for i, rate := range firstRateArray {

		ratesDto[i] = RateDto{
			ID:         *rate.ID,
			Key:        *api.Encode(rate.ID),
			Rate:       rate.Rate,
			DateOfRate: rate.DateOfRate.Format(time.DateOnly),
		}
	}

	for i, rate := range secondRateArray {

		ratesDto[i+firstLen] = RateDto{
			ID:         *rate.ID,
			Key:        *api.Encode(rate.ID),
			Rate:       rate.Rate,
			DateOfRate: rate.DateOfRate.Format(time.DateOnly),
		}
	}

	return ratesDto, nil

}

func deleteReceipt(keys Keys) (int64, error) {
	numWorkers := 4
	var wg sync.WaitGroup
	resultChan := make(chan int64, numWorkers)
	errorChan := make(chan error, numWorkers)

	wg.Add(numWorkers)

	go func() {
		defer wg.Done()
		rowsAffected, err := deleteById(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting receipt: %s %d -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := extraCharges.DeleteByReceipt(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting extra charges: %s %d -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := expenses.DeleteByReceipt(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting expenses: %s %d -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := debts.DeleteByReceipt(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting debts: %s %d -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	wg.Wait()
	close(resultChan)
	close(errorChan)

	multiErr := &util.MultiError{Errors: make([]error, len(errorChan))}
	for err := range errorChan {
		multiErr.Add(err)
	}

	if multiErr.HasErrors() {
		return 0, multiErr
	}

	var sum int64 = 0
	for value := range resultChan {
		sum += value
	}

	log.Printf("Deleted %d records", sum)

	return sum, nil
}
