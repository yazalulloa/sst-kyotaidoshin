package receipts

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/debts"
	"github.com/yaz/kyo-repo/internal/expenses"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/reserveFunds"
	"github.com/yaz/kyo-repo/internal/util"
	"golang.org/x/sync/syncmap"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) loadRates(records []ReceiptRecord, ratesHolder *RatesHolder) error {
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

	var wg sync.WaitGroup
	wg.Add(len(ratesToLookUp))
	errorChan := make(chan error, len(ratesToLookUp))

	for _, date := range ratesToLookUp {

		go func(date string) {
			defer wg.Done()

			parsedDate, err := time.Parse(time.DateOnly, date)
			if err != nil {
				errorChan <- fmt.Errorf("error parsing date %s: %w", date, err)
				return
			}

			rate, err := rates.NewRepository(service.repo.ctx).GetFirstBeforeDate(util.USD.Name(), parsedDate)
			if err != nil {
				errorChan <- fmt.Errorf("error getting rate for date %s: %w", date, err)
				return
			}

			ratesHolder.Rates.Store(date, *rate.ID)
		}(date)
	}

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return fmt.Errorf("error loading rates: %v", err)
	}

	return nil
}

func (service Service) insertRecord(records []ReceiptRecord, ratesHolder *RatesHolder) (int64, error) {

	err := service.loadRates(records, ratesHolder)
	if err != nil {
		return 0, err
	}

	receiptsArray := make([]model.Receipts, len(records))
	var extraChargeArray []model.ExtraCharges
	var expensesArray []model.Expenses
	var debtsArray []model.Debts

	counter := int64(len(records))
	for i, record := range records {

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

		receiptId := util.UuidV7()

		receipt := model.Receipts{
			ID:         receiptId,
			BuildingID: record.Receipt.BuildingID,
			Year:       record.Receipt.Year,
			Month:      record.Receipt.Month,
			Date:       date,
			RateID:     rateId.(int64),
			Sent:       record.Receipt.Sent,
			LastSent:   lastSent,
			CreatedAt:  &createdAt,
		}

		receiptsArray[i] = receipt

		for _, extraCharge := range record.ExtraCharges {
			var builder strings.Builder
			for idx, apt := range extraCharge.Apartments {
				builder.WriteString(apt.Number)
				if idx < len(extraCharge.Apartments)-1 {
					builder.WriteString(",")
				}
			}

			extraChargeArray = append(extraChargeArray, model.ExtraCharges{
				BuildingID:      extraCharge.BuildingID,
				ParentReference: receiptId,
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
				ReceiptID:   receiptId,
				Description: expense.Description,
				Amount:      expense.Amount,
				Currency:    expense.Currency,
				Type:        expense.Type,
			})

		}

		for _, debt := range record.Debts {

			years := make([]debts.YearWithMonths, 0)

			if len(debt.Months) > 0 {
				years = append(years, debts.YearWithMonths{
					Year:   record.Receipt.Year,
					Months: debt.Months,
				})
			}

			monthlyDebt := debts.MonthlyDebt{
				Amount: 0,
				Years:  years,
			}

			bytes, err := json.Marshal(monthlyDebt)
			if err != nil {
				return 0, err
			}

			debtsArray = append(debtsArray, model.Debts{
				BuildingID:                    debt.BuildingID,
				ReceiptID:                     receiptId,
				AptNumber:                     debt.AptNumber,
				Receipts:                      debt.Receipts,
				Amount:                        debt.Amount,
				Months:                        string(bytes),
				PreviousPaymentAmount:         debt.PreviousPaymentAmount,
				PreviousPaymentAmountCurrency: debt.PreviousPaymentAmountCurrency,
			})
		}

	}

	slices.SortFunc(extraChargeArray, func(a, b model.ExtraCharges) int {
		return cmp.Or(
			cmp.Compare(a.Description, b.Description),
			cmp.Compare(a.Amount, b.Amount),
		)
	})

	var wg sync.WaitGroup
	wg.Add(4)
	errorChan := make(chan error, 4)

	go func() {
		defer wg.Done()
		_, err := service.repo.InsertBulk(receiptsArray)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		rows, err := extraCharges.NewRepository(service.repo.ctx).InsertBulk(extraChargeArray)
		if err != nil {
			errorChan <- err
			return
		}
		log.Printf("Inserted %d/%d extra charges", len(extraChargeArray), rows)
	}()

	go func() {
		defer wg.Done()
		rows, err := expenses.NewRepository(service.repo.ctx).InsertBulk(expensesArray)
		if err != nil {
			errorChan <- err
			return
		}
		log.Printf("Inserted %d/%d expenses", len(expensesArray), rows)
	}()

	go func() {
		defer wg.Done()

		rows, err := debts.NewRepository(service.repo.ctx).InsertBulk(debtsArray)
		if err != nil {
			errorChan <- err
			return
		}
		log.Printf("Inserted %d/%d debts", len(debtsArray), rows)
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return 0, err
	}

	return counter, nil
}

func (service Service) getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {

	//start := time.Now()
	//defer func() { log.Printf("Elapsed time getTableResponse receipts: %v\n", time.Since(start)) }()

	var tableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		//start := time.Now()
		//defer func() { log.Printf("Elapsed time selectList: %v\n", time.Since(start)) }()

		array, err := service.repo.selectList(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}

		items := make([]Item, len(array))
		for i, item := range array {

			obj, err := toItem(&item, nil)
			if err != nil {
				errorChan <- err
				return
			}

			items[i] = *obj

		}
		tableResponse.Results = items
	}()

	go func() {
		defer wg.Done()
		//start := time.Now()
		//defer func() { log.Printf("Elapsed time getTotalCount: %v\n", time.Since(start)) }()

		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		//start := time.Now()
		//defer func() { log.Printf("Elapsed time getQueryCount: %v\n", time.Since(start)) }()

		queryCount, err := service.repo.getQueryCount(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	return &tableResponse, nil
}

func toItem(item *model.Receipts, oldCardId *string) (*Item, error) {
	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(*item, cardIdStr)
	key := *util.Encode(keys)

	var lastSent *int64
	if item.LastSent != nil {
		tmp := item.LastSent.UnixMilli()
		lastSent = &tmp
	}

	params := UpdateParams{
		Key:      key,
		Building: item.BuildingID,
		Year:     item.Year,
		Month:    item.Month,
		Date:     item.Date.Format(time.DateOnly),
	}

	byteArray, err := json.Marshal(params)

	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	return &Item{
		CardId:       keys.CardId,
		Key:          key,
		Item:         *item,
		CreatedAt:    item.CreatedAt.UnixMilli(),
		LastSent:     lastSent,
		UpdateParams: base64Str,
	}, nil
}

func (service Service) getItem(id string, oldCardId *string) (*Item, error) {
	receipt, err := service.repo.selectById(id)
	if err != nil {
		return nil, err
	}

	item, err := toItem(receipt, oldCardId)
	if err != nil {
		return nil, err
	}

	if receipt == nil {
		return nil, errors.New("receipt not found")
	}

	return item, nil
}

func (service Service) getFormDto(keys Keys) (*FormDto, error) {
	formDto := FormDto{}

	var wg sync.WaitGroup
	wg.Add(7)
	errorChan := make(chan error, 7)

	go func() {
		defer wg.Done()

		building, err := buildings.NewRepository(service.repo.ctx).SelectById(keys.BuildingId)
		if err != nil {
			errorChan <- err
			return
		}

		formDto.building = *building
	}()

	go func() {
		defer wg.Done()

		receipt, err := service.repo.selectById(keys.Id)
		if err != nil {
			errorChan <- err
			return
		}

		if receipt == nil {
			errorChan <- fmt.Errorf("receipt not found: %s", keys.Id)
			return
		}

		ratesDtos, err := service.getRatesDtos(&receipt.Date)
		if err != nil {
			errorChan <- err
			return
		}

		newKeys := Keys{
			BuildingId: receipt.BuildingID,
			Id:         receipt.ID,
		}

		newKeysStr := util.Encode(newKeys)

		updateParams := UpdateParams{
			Key:   *newKeysStr,
			Year:  receipt.Year,
			Month: receipt.Month,
			Date:  receipt.Date.Format(time.DateOnly),
		}

		byteArray, err := json.Marshal(updateParams)

		if err != nil {
			errorChan <- err
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

		dto, err := expenses.NewRepository(service.repo.ctx).GetFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- err
			return
		}

		formDto.expenseFormDto = *dto
	}()

	go func() {
		defer wg.Done()

		reserveFundFormDto, err := reserveFunds.NewService(service.repo.ctx).GetFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- err
			return
		}

		formDto.reserveFundFormDto = *reserveFundFormDto
	}()

	go func() {
		defer wg.Done()

		dto, err := extraCharges.NewService(service.repo.ctx).GetReceiptFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- err
			return
		}

		formDto.extraChargesFormDto = *dto
	}()

	go func() {
		defer wg.Done()

		dto, err := debts.NewRepository(service.repo.ctx).GetFormDto(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- err
			return
		}

		formDto.debtFormDto = *dto
	}()

	go func() {
		defer wg.Done()
		apts, err := apartments.NewRepository(service.repo.ctx).SelectNumberAndNameByBuildingId(keys.BuildingId)
		if err != nil {
			errorChan <- err
			return
		}

		aptStr, err := json.Marshal(apts)
		if err != nil {
			errorChan <- err
			return
		}

		base64Str := base64.URLEncoding.EncodeToString(aptStr)

		formDto.apts = base64Str
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
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

func (service Service) getRatesDtos(date *time.Time) ([]RateDto, error) {

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	var firstRateArray []model.Rates
	var secondRateArray []model.Rates

	go func() {
		defer wg.Done()
		arr, err := rates.NewRepository(service.repo.ctx).GetFromDate(util.USD.Name(), *date, 5, true)

		if err != nil {
			errorChan <- err
			return
		}

		slices.Reverse(arr)
		firstRateArray = arr
	}()

	go func() {
		defer wg.Done()
		arr, err := rates.NewRepository(service.repo.ctx).GetFromDate(util.USD.Name(), *date, 5, false)

		if err != nil {
			errorChan <- err
			return
		}

		secondRateArray = arr
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	firstLen := len(firstRateArray)
	ratesDto := make([]RateDto, len(firstRateArray)+len(secondRateArray))

	for i, rate := range firstRateArray {

		ratesDto[i] = RateDto{
			ID:         *rate.ID,
			Key:        *util.Encode(rate.ID),
			Rate:       rate.Rate,
			DateOfRate: rate.DateOfRate.Format(time.DateOnly),
		}
	}

	for i, rate := range secondRateArray {

		ratesDto[i+firstLen] = RateDto{
			ID:         *rate.ID,
			Key:        *util.Encode(rate.ID),
			Rate:       rate.Rate,
			DateOfRate: rate.DateOfRate.Format(time.DateOnly),
		}
	}

	return ratesDto, nil

}

func (service Service) deleteReceipt(keys Keys) (int64, error) {
	numWorkers := 4
	var wg sync.WaitGroup
	resultChan := make(chan int64, numWorkers)
	errorChan := make(chan error, numWorkers)

	wg.Add(numWorkers)

	go func() {
		defer wg.Done()
		rowsAffected, err := service.repo.deleteById(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting receipt: %s %s -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := extraCharges.NewRepository(service.repo.ctx).DeleteByReceipt(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting extra charges: %s %s -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := expenses.NewRepository(service.repo.ctx).DeleteByReceipt(keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting expenses: %s %s -> %w", keys.BuildingId, keys.Id, err)
			return
		}

		resultChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := debts.NewRepository(service.repo.ctx).DeleteByReceipt(keys.BuildingId, keys.Id)
		if err != nil {
			errorChan <- fmt.Errorf("error deleting debts: %s %s -> %w", keys.BuildingId, keys.Id, err)
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

func (service Service) duplicate(key Keys) (*string, error) {
	var wg sync.WaitGroup
	wg.Add(4)
	errorChan := make(chan error, 4)

	var receipt *model.Receipts
	var debtArray []model.Debts
	var expenseArray []model.Expenses
	var extraChargeArray []model.ExtraCharges

	go func() {
		defer wg.Done()
		rec, err := service.repo.selectById(key.Id)
		if err != nil {
			errorChan <- err
			return
		}
		receipt = rec
	}()

	go func() {
		defer wg.Done()
		array, err := debts.NewRepository(service.repo.ctx).SelectByBuildingReceipt(key.BuildingId, key.Id)
		if err != nil {
			errorChan <- err
			return
		}
		debtArray = array
	}()

	go func() {
		defer wg.Done()
		array, err := expenses.NewRepository(service.repo.ctx).SelectByReceipt(key.Id)
		if err != nil {
			errorChan <- err
			return
		}
		expenseArray = array
	}()

	go func() {
		defer wg.Done()
		array, err := extraCharges.NewRepository(service.repo.ctx).SelectByReceipt(key.Id)
		if err != nil {
			errorChan <- err
			return
		}
		extraChargeArray = array
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	receipt.ID = util.UuidV7()
	receipt.Sent = false

	for i := range debtArray {
		debtArray[i].ReceiptID = receipt.ID
	}

	for i := range expenseArray {
		expenseArray[i].ReceiptID = receipt.ID
	}

	for i := range extraChargeArray {
		extraChargeArray[i].ParentReference = receipt.ID
	}

	rowsChan := make(chan int64, 4)
	errorChan = make(chan error, 4)
	wg.Add(4)

	go func() {
		defer wg.Done()

		rowsAffected, err := service.repo.insert(*receipt)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := debts.NewRepository(service.repo.ctx).InsertBulk(debtArray)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := expenses.NewRepository(service.repo.ctx).InsertBulk(expenseArray)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := extraCharges.NewRepository(service.repo.ctx).InsertBulk(extraChargeArray)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	wg.Wait()
	close(errorChan)
	close(rowsChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	var sum int64 = 0
	for value := range rowsChan {
		sum += value
	}

	log.Printf("Inserted %d records", sum)

	return util.Encode(keys(*receipt, "")), nil
}

func (service Service) Backup() (string, error) {

	requestQuery := RequestQuery{
		Limit:     30,
		SortOrder: util.SortOrderTypeASC,
	}
	selectListDtos := func() ([]ReceiptBackup, error) {
		list, err := service.repo.selectList(requestQuery)
		if err != nil {
			return nil, err
		}

		ids := make([]string, len(list))

		for i, item := range list {
			ids[i] = item.ID

			if i == len(list)-1 {
				requestQuery.LastId = item.ID
			}

		}

		var extraChargesArray []model.ExtraCharges
		var expensesArray []model.Expenses
		var debtsArray []model.Debts

		var wg sync.WaitGroup
		wg.Add(3)
		errorChan := make(chan error, 3)

		go func() {
			defer wg.Done()

			extraChargesArray, err = extraCharges.NewRepository(service.repo.ctx).SelectByReceipts(ids)

			if err != nil {
				errorChan <- err
				return
			}
		}()

		go func() {
			defer wg.Done()

			expensesArray, err = expenses.NewRepository(service.repo.ctx).SelectByReceipts(ids)
			if err != nil {
				errorChan <- err
				return
			}

		}()

		go func() {
			defer wg.Done()

			debtsArray, err = debts.NewRepository(service.repo.ctx).SelectByReceipts(ids)
			if err != nil {
				errorChan <- err
				return
			}

		}()

		wg.Wait()
		close(errorChan)

		err = util.HasErrors(errorChan)

		if err != nil {
			return nil, err
		}

		dtos := make([]ReceiptBackup, len(list))

		for i, receipt := range list {

			extraChargesBackup := make([]extraCharges.ExtraChargeDto, 0)

			for _, extraCharge := range extraChargesArray {

				if extraCharge.ParentReference == receipt.ID {
					apts := strings.Split(extraCharge.Apartments, ",")

					aptsDto := make([]extraCharges.AptDto, len(apts))
					for k, apt := range apts {
						aptsDto[k] = extraCharges.AptDto{
							Number: apt,
						}
					}

					extraChargesBackup = append(extraChargesBackup, extraCharges.ExtraChargeDto{
						BuildingID:      extraCharge.BuildingID,
						ParentReference: extraCharge.ParentReference,
						Type:            extraCharge.Type,
						Description:     extraCharge.Description,
						Amount:          extraCharge.Amount,
						Currency:        extraCharge.Currency,
						Active:          extraCharge.Active,
						Apartments:      aptsDto,
					})

				}

			}

			expensesBackup := make([]expenses.ExpenseBackup, 0)

			for _, expense := range expensesArray {
				if expense.ReceiptID == receipt.ID {
					expensesBackup = append(expensesBackup, expenses.ExpenseBackup{
						BuildingID:  expense.BuildingID,
						ReceiptID:   expense.ReceiptID,
						Description: expense.Description,
						Amount:      expense.Amount,
						Currency:    expense.Currency,
						Type:        expense.Type,
					})
				}
			}

			debtsBackup := make([]debts.DebtBackup, 0)

			for _, debt := range debtsArray {
				if debt.ReceiptID == receipt.ID {
					debtsBackup = append(debtsBackup, debts.DebtBackup{
						BuildingID:                    debt.BuildingID,
						ReceiptID:                     debt.ReceiptID,
						AptNumber:                     debt.AptNumber,
						Receipts:                      debt.Receipts,
						Amount:                        debt.Amount,
						PreviousPaymentAmount:         debt.PreviousPaymentAmount,
						PreviousPaymentAmountCurrency: debt.PreviousPaymentAmountCurrency,
					})
				}
			}

			lastSent := ""

			if receipt.LastSent != nil {
				lastSent = receipt.LastSent.Format(time.RFC3339)
			}

			dtos[i] = ReceiptBackup{
				Receipt: ReceiptDto{
					BuildingID: receipt.BuildingID,
					Year:       receipt.Year,
					Month:      receipt.Month,
					Date:       receipt.Date.Format(time.DateOnly),
					Sent:       receipt.Sent,
					LastSent:   &lastSent,
				},
				ExtraCharges: extraChargesBackup,
				Expenses:     expensesBackup,
				Debts:        debtsBackup,
			}
		}

		return dtos, nil
	}

	return api.Backup(api.BACKUP_RECEIPTS_FILE, selectListDtos)
}

func (service Service) ProcessDecoder(decoder *json.Decoder) (int64, error) {
	var records []ReceiptRecord
	err := decoder.Decode(&records)
	if err != nil {
		log.Printf("Error decoding json: %s", err)
		return 0, err
	}

	slices.SortFunc(records, func(a, b ReceiptRecord) int {

		lhs, err := time.Parse(time.DateOnly, a.Receipt.Date)
		if err != nil {
			//panic(err)
			log.Printf("Error parsing date: %s %v", a.Receipt.Date, err)
			return 0
		}

		rhs, err := time.Parse(time.DateOnly, b.Receipt.Date)
		if err != nil {
			//panic(err)
			log.Printf("Error parsing date: %s %v", b.Receipt.Date, err)
			return 0
		}

		return lhs.Compare(rhs)
	})

	array := util.SplitArray(records, 15)

	var total int64
	ratesHolder := RatesHolder{Rates: syncmap.Map{}}
	for _, chunk := range array {
		rowsAffected, err := service.insertRecord(chunk, &ratesHolder)
		if err != nil {
			return 0, err
		}
		total += rowsAffected
	}

	return total, nil
}
