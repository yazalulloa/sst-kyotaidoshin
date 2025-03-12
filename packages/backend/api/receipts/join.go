package receipts

import (
	"db/gen/model"
	"encoding/json"
	"errors"
	"fmt"
	"kyotaidoshin/apartments"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"log"
	"slices"
	"strings"
	"sync"
)

func JoinExpensesAndReserveFunds(buildingId string, receiptId string) (*expenses.ReceiptExpensesDto, error) {
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
		dto, err := reserveFunds.GetFormDto(buildingId, receiptId)
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

func GetReceiptExpensesDto(receiptId string, expenseArray []expenses.Item, reserveFundArray []reserveFunds.Item) expenses.ReceiptExpensesDto {
	totals := expenses.ExpenseTotals{}

	totalCommon, totalUnCommon := expenses.Totals(expenseArray)

	totals.TotalCommon = totalCommon
	totals.TotalUnCommon = totalUnCommon

	reserveFundExpenses := make([]expenses.Item, 0)

	isTherePercentage := false

	for _, item := range reserveFundArray {
		if item.Item.Active && item.Item.AddToExpenses {

			var total float64

			if expenses.COMMON.Is(item.Item.ExpenseType) {
				total = totalCommon
			} else {
				total = totalUnCommon
			}

			var amount float64
			nameSuffix := ""
			if reserveFunds.FIXED_PAY.FundIs(item.Item) {
				amount = item.Item.Pay
			} else {
				amount = util.PercentageOf(item.Item.Pay, total)
				amount = util.RoundFloat(amount, 2)
				nameSuffix = " " + util.FormatFloat2(item.Item.Pay) + "%"
				isTherePercentage = true
			}

			expenseItem := expenses.Item{
				CardId: cardId(),
				Item: model.Expenses{
					BuildingID:  item.Item.BuildingID,
					ReceiptID:   receiptId,
					Description: item.Item.Name + nameSuffix,
					Amount:      amount,
					Currency:    util.VED.Name(),
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

	return expenses.ReceiptExpensesDto{
		IsTherePercentage:   isTherePercentage,
		ReserveFundExpenses: reserveFundExpenses,
		Totals:              totals,
	}
}

func CalculateReceipt(buildingId string, receiptId string) (*CalculatedReceipt, error) {

	var wg sync.WaitGroup
	wg.Add(8)
	errorChan := make(chan error, 8)

	calculatedReceipt := CalculatedReceipt{}

	var reserveFundArray []model.ReserveFunds
	var apartmentArray []model.Apartments
	var debtArray []model.Debts
	var buildingExtraCharges []model.ExtraCharges
	var receiptExtraCharges []model.ExtraCharges

	go func() {
		defer wg.Done()

		item, err := selectByIdWithRate(receiptId)
		if err != nil {
			errorChan <- err
			return
		}

		calculatedReceipt.Receipt = item.Receipts

		calculatedReceipt.MonthStr = util.FromInt16ToMonth(item.Receipts.Month)
		calculatedReceipt.Rate = item.Rates
	}()

	go func() {
		defer wg.Done()

		building, err := buildings.SelectById(buildingId)
		if err != nil {
			errorChan <- err
			return
		}

		calculatedReceipt.Building = *building
		split := strings.Split(building.CurrenciesToShowAmountToPay, ",")
		currencies := make([]util.AllowedCurrencies, len(split))
		for i, v := range split {
			currencies[i] = util.GetAllowedCurrency(v)
		}
		calculatedReceipt.CurrenciesToShowAmountToPay = currencies

	}()

	go func() {
		defer wg.Done()

		array, err := reserveFunds.SelectByBuilding(buildingId)
		if err != nil {
			errorChan <- err
			return
		}

		reserveFundArray = array
	}()

	go func() {
		defer wg.Done()

		array, err := apartments.SelectByBuilding(buildingId)
		if err != nil {
			errorChan <- err
			return
		}

		apartmentArray = array
	}()

	go func() {
		defer wg.Done()
		array, err := expenses.SelectByReceipt(receiptId)
		if err != nil {
			errorChan <- err
			return
		}

		expensesArray := make([]ExpenseAttr, 0)
		for _, expense := range array {
			if expense.Description != expenses.AliquotDifference {
				expensesArray = append(expensesArray, ExpenseAttr{
					Expense: expense,
				})
			}
		}

		calculatedReceipt.Expenses = expensesArray
	}()

	go func() {
		defer wg.Done()

		array, err := debts.SelectByBuildingReceipt(buildingId, receiptId)
		if err != nil {
			errorChan <- err
			return
		}

		debtArray = array

		var receiptsAmount int16 = 0
		var debtTotal float64 = 0
		for _, debt := range debtArray {
			receiptsAmount += debt.Receipts
			debtTotal += debt.Amount
		}

		calculatedReceipt.DebtReceiptsAmount = receiptsAmount
		calculatedReceipt.DebtTotal = util.RoundFloat(debtTotal, 2)
	}()

	go func() {
		defer wg.Done()

		array, err := extraCharges.SelectByReceipt(receiptId)
		if err != nil {
			errorChan <- err
			return
		}

		receiptExtraCharges = array
	}()

	go func() {
		defer wg.Done()

		array, err := extraCharges.SelectByBuilding(buildingId)
		if err != nil {
			errorChan <- err
			return
		}

		buildingExtraCharges = array
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	usdRate := calculatedReceipt.Rate.Rate

	toUsd := func(ved float64) float64 {
		return ved / usdRate
	}

	toVed := func(usd float64) float64 {
		return usd * usdRate
	}

	commonTotalBeforeFund := ExpenseAmountsByCurrency(calculatedReceipt.Expenses, usdRate, expenses.COMMON.ExpenseIs)

	fundWithAmounts := make([]ReserveFundWithCalculatedAmount, 0)

	debtTableAdded := false
	debtCurrency := util.GetAllowedCurrency(calculatedReceipt.Building.DebtCurrency)
	for _, fund := range reserveFundArray {
		if fund.Expense != 0 {
			calculatedReceipt.ThereIsReserveFundExpense = true
		}

		if fund.Active && fund.Pay > 0 {
			var newAmount float64
			if reserveFunds.FIXED_PAY.FundIs(fund) {
				newAmount = fund.Pay
			} else {
				newAmount = util.PercentageOf(fund.Pay, commonTotalBeforeFund.Amount)
				newAmount = util.RoundFloat(newAmount, 2)
			}

			if fund.AddToExpenses {
				symbol := ""

				if reserveFunds.PERCENTAGE.FundIs(fund) {
					symbol = "%"
				}

				calculatedReceipt.Expenses = append(calculatedReceipt.Expenses, ExpenseAttr{
					Expense: model.Expenses{
						Description: fmt.Sprintf("%s %s%s", fund.Name, util.FormatFloat64(fund.Pay), symbol),
						Amount:      newAmount,
						Currency:    commonTotalBeforeFund.Currency.Name(),
						Type:        fund.ExpenseType,
					},
					IsReserveFund: true,
				})
			}

			newFund := fund.Fund + newAmount
			if fund.Expense != 0 {
				newFund -= fund.Expense
			}

			var amountToPay string
			if reserveFunds.PERCENTAGE.FundIs(fund) {
				amountToPay = fmt.Sprintf("%s %s%s", util.VED.Format(newAmount), util.FormatFloat64(fund.Pay), "%")
			} else {
				amountToPay = util.VED.Format(newAmount)
			}

			fundWithAmounts = append(fundWithAmounts, ReserveFundWithCalculatedAmount{
				Amount:           newAmount,
				Fund:             fund,
				FundFormatted:    util.VED.Format(fund.Fund),
				AmountToPay:      amountToPay,
				ExpenseFormatted: util.VED.Format(fund.Expense),
				NewReserveFund:   util.VED.Format(newFund),
			})

			if fund.Name == "FONDO DE RESERVA" || fund.Name == "FONDO/RESERVA" && !debtTableAdded {
				debtTableAdded = true

				modifiedDebt := calculatedReceipt.DebtTotal

				if debtCurrency != util.VED {
					modifiedDebt = toVed(calculatedReceipt.DebtTotal)
				}

				modifiedDebt -= newFund
				modifiedDebt = util.RoundFloat(modifiedDebt, 2)

				fundWithAmounts = append(fundWithAmounts, ReserveFundWithCalculatedAmount{
					Fund: model.ReserveFunds{
						Name: fmt.Sprintf("P/Cobrar > Recibos  %d", calculatedReceipt.DebtReceiptsAmount),
					},
					FundFormatted:  debtCurrency.Format(calculatedReceipt.DebtTotal),
					AmountToPay:    "DEFICIT/Patrimonio",
					NewReserveFund: util.VED.Format(modifiedDebt),
				})

			}
		}

	}

	calculatedReceipt.ReserveFunds = fundWithAmounts

	commonTotal := ExpenseAmountsByCurrency(calculatedReceipt.Expenses, usdRate, expenses.COMMON.ExpenseIs)
	commonTotal.Amount = util.RoundFloat(commonTotal.Amount, 2)
	calculatedReceipt.TotalCommonExpenses = commonTotal.Amount
	calculatedReceipt.TotalCommonExpensesCurrency = commonTotal.Currency

	var aliquotDifference float64 = 0

	if commonTotal.Amount != 0 {

		var totalAliquot float64 = 0

		for _, apt := range apartmentArray {
			totalAliquot += util.PercentageOf(apt.Aliquot, commonTotal.Amount)
		}

		diff := totalAliquot - commonTotal.Amount
		diff = util.RoundFloat(diff, 2)
		if diff > 0 {
			aliquotDifference = diff
		}
	}

	calculatedReceipt.Expenses = append(calculatedReceipt.Expenses, ExpenseAttr{
		Expense: model.Expenses{
			Description: expenses.AliquotDifference,
			Amount:      aliquotDifference,
			Currency:    commonTotal.Currency.Name(),
			Type:        expenses.UNCOMMON.Name(),
		},
	})

	uncommonTotal := ExpenseAmountsByCurrency(calculatedReceipt.Expenses, usdRate, expenses.UNCOMMON.ExpenseIs)
	calculatedReceipt.TotalUnCommonExpenses = uncommonTotal.Amount
	calculatedReceipt.TotalUnCommonExpensesCurrency = uncommonTotal.Currency

	var unCommonPay float64 = 0

	if uncommonTotal.Amount != 0 {
		unCommonPay = uncommonTotal.Amount / float64(len(apartmentArray))
	}

	aptTotals := make([]AptTotal, 0)
	var aptTotal float64 = 0
	for _, apt := range apartmentArray {

		aptExtraCharges := make([]model.ExtraCharges, 0)

		var preCalculatedPayment float64
		if calculatedReceipt.Building.FixedPay {
			preCalculatedPayment = calculatedReceipt.Building.FixedPayAmount
		} else {
			preCalculatedPayment = util.PercentageOf(apt.Aliquot, commonTotal.Amount) + unCommonPay
			preCalculatedPayment = util.RoundFloat(preCalculatedPayment, 2)
		}

		var vedExtraCharge float64 = 0
		var usdExtraCharge float64 = 0

		for _, extraCharge := range slices.Concat(buildingExtraCharges, receiptExtraCharges) {
			if extraCharge.Active && extraCharge.Amount > 0 {
				split := strings.Split(extraCharge.Apartments, ",")
				if slices.Contains(split, apt.Number) {

					aptExtraCharges = append(aptExtraCharges, extraCharge)

					switch extraCharge.Currency {
					case util.USD.Name():
						usdExtraCharge += extraCharge.Amount
						break
					case util.VED.Name():
						vedExtraCharge += extraCharge.Amount
						break
					default:
						log.Printf("Unknown currency: %s\n", extraCharge.Currency)
						return nil, errors.New(fmt.Sprintf("Unknown currency %s", extraCharge.Currency))
					}

				}

			}
		}

		var usdPay = usdExtraCharge
		var vedPay = vedExtraCharge

		if usdExtraCharge > 0 {
			vedPay += toVed(usdExtraCharge)
		}

		if vedExtraCharge > 0 {
			usdPay += toUsd(vedExtraCharge)
		}

		if calculatedReceipt.Building.FixedPay {
			if util.USD.Is(calculatedReceipt.Building.MainCurrency) {
				usdPay += preCalculatedPayment
				vedPay += toVed(preCalculatedPayment)
			} else {
				usdPay += toUsd(preCalculatedPayment)
				vedPay += preCalculatedPayment
			}

		} else {
			usdPay += toUsd(preCalculatedPayment)
			vedPay += preCalculatedPayment
		}

		debt := model.Debts{
			PreviousPaymentAmountCurrency: calculatedReceipt.Building.DebtCurrency,
		}

		for _, d := range debtArray {
			if d.AptNumber == apt.Number {
				debt = d
				break
			}
		}

		debtMonthStr := "SOLVENTE"
		if debt.Months != "" {
			var monthlyDebt debts.MonthlyDebt
			err := json.Unmarshal([]byte(debt.Months), &monthlyDebt)
			if err != nil {
				return nil, err
			}

			if monthlyDebt.Amount > 0 {
				debtMonthStr = fmt.Sprintf("%d MESES", monthlyDebt.Amount)
			} else if len(monthlyDebt.Years) > 0 {

				var builder strings.Builder
				for i, year := range monthlyDebt.Years {

					if len(year.Months) > 0 {
						if year.Year > 0 {
							builder.WriteString(fmt.Sprintf("%d: ", year.Year))
						}
						for j, month := range year.Months {

							builder.WriteString(util.FromInt16ToMonth(month))
							if j != len(year.Months)-1 {
								builder.WriteString(", ")
							}
						}
						if i != len(monthlyDebt.Years)-1 {
							builder.WriteString(" - ")
						}
					}
				}

				debtMonthStr = builder.String()
			}
		}

		amounts := []AmountWithCurrency{
			{
				Amount:   util.RoundFloat(vedPay, 2),
				Currency: util.VED,
			},
			{
				Amount:   util.RoundFloat(usdPay, 2),
				Currency: util.USD,
			},
		}

		for _, v := range amounts {
			if v.Currency.Is(calculatedReceipt.Building.MainCurrency) {
				aptTotal += v.Amount
			}
		}

		aptTotals = append(aptTotals, AptTotal{
			Apartment:    apt,
			Amounts:      amounts,
			ExtraCharges: aptExtraCharges,
			Debt:         debt,
			DebtMonthStr: debtMonthStr,
		})

	}

	calculatedReceipt.Apartments = aptTotals
	calculatedReceipt.ApartmentsTotal = util.RoundFloat(aptTotal, 2)

	return &calculatedReceipt, err
}
