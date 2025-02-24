package receipts

import (
	"db/gen/model"
	"kyotaidoshin/expenses"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"sync"
)

func JoinExpensesAndReserveFunds(buildingId string, receiptId int32) (*ReceiptExpensesDto, error) {
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

	var fundFormDto *reserveFunds.FormDto
	var expenseFormDto *expenses.FormDto

	go func() {
		defer wg.Done()
		dto, err := reserveFunds.GetFormDto(buildingId)
		if err != nil {
			handleErr(err)
			return
		}

		fundFormDto = dto
	}()

	go func() {
		defer wg.Done()
		dto, err := expenses.GetFormDto(buildingId, receiptId)
		if err != nil {
			handleErr(err)
			return
		}
		expenseFormDto = dto
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	dto := GetReceiptExpensesDto(receiptId, expenseFormDto.Items, fundFormDto.Items)
	return &dto, nil
}

func GetReceiptExpensesDto(receiptId int32, expenseArray []expenses.Item, reserveFundArray []reserveFunds.Item) ReceiptExpensesDto {
	totals := ExpenseTotals{}

	totalCommon, totalUnCommon := expenses.Totals(expenseArray)

	totals.TotalCommon = totalCommon
	totals.TotalUnCommon = totalUnCommon

	reserveFundExpenses := make([]expenses.Item, 0)

	for _, item := range reserveFundArray {
		if item.Item.Active && item.Item.AddToExpenses {

			var total float64
			if item.Item.Type == "COMMON" {
				total = totalCommon
			} else {
				total = totalUnCommon
			}

			var amount float64
			nameSuffix := ""
			if reserveFunds.IsFixedPay(item.Item.Type) {
				amount = item.Item.Pay
			} else {
				amount = util.PercentageOf(item.Item.Pay, total)
				nameSuffix = " " + util.FormatFloat2(item.Item.Pay) + "%"
			}

			expenseItem := expenses.Item{
				CardId: cardId(),
				Item: model.Expenses{
					BuildingID:  item.Item.BuildingID,
					ReceiptID:   receiptId,
					Description: item.Item.Name + nameSuffix,
					Amount:      amount,
					Currency:    "VED",
					Type:        item.Item.ExpenseType,
				},
			}

			reserveFundExpenses = append(reserveFundExpenses, expenseItem)
		}
	}

	joinArray := append(reserveFundExpenses, expenseArray...)
	totalCommonPlusReserve, totalUnCommonPlusReserve := expenses.Totals(joinArray)

	totals.TotalCommonPlusReserve = totalCommonPlusReserve
	totals.TotalUnCommonPlusReserve = totalUnCommonPlusReserve
	totals.ExpensesCounter = len(joinArray)

	return ReceiptExpensesDto{
		reserveFundExpenses: reserveFundExpenses,
		totals:              totals,
	}
}
