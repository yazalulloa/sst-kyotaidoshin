package receipts

import (
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"errors"
	"kyotaidoshin/api"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"log"
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

			rate, err := rates.GetFirstBeforeDate(parsedDate)
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
				ReserveFund: expense.ReserveFund,
				Type:        expense.Type,
			})

		}

		for _, debt := range record.Debts {
			stringArray := make([]string, len(debt.Months))

			for i, num := range debt.Months {
				stringArray[i] = strconv.Itoa(int(num))
			}
			months := strings.Join(stringArray, ",")
			debtArray := model.Debts{
				BuildingID: debt.BuildingID,
				ReceiptID:  int32(receiptId),
				AptNumber:  debt.AptNumber,
				Amount:     debt.Amount,
				Months:     months,
			}
			debtsArray = append(debtsArray, debtArray)
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
		_, err := extraCharges.InsertBackup(extraChargeArray)
		handleErr(err)
	}()

	go func() {
		defer wg.Done()
		_, err := expenses.InsertBackup(expensesArray)
		handleErr(err)
	}()

	go func() {
		defer wg.Done()
		_, err := debts.InsertBackup(debtsArray)
		handleErr(err)
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

	updateParams := UpdateParams{
		Key:    key,
		Year:   item.Year,
		Month:  item.Month,
		Date:   item.Date,
		RateID: item.RateID,
	}

	byteArray, err := json.Marshal(updateParams)

	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	return &Item{
		CardId:       keys.CardId,
		Key:          key,
		Item:         *item,
		CreatedAt:    item.CreatedAt.UnixMilli(),
		UpdateParams: &base64Str,
	}, nil
}
