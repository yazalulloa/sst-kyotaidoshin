package receipts

import (
	"db/gen/model"
	"kyotaidoshin/util"
)

type AmountWithCurrency struct {
	Amount   float64
	Currency util.AllowedCurrencies
}

func ExpenseAmountsByCurrency(array []ExpenseAttr, usdRate float64, predicate func(str model.Expenses) bool) AmountWithCurrency {

	var usdAmount float64 = 0
	var vedAmount float64 = 0

	for _, expense := range array {
		if predicate(expense.Expense) {
			if util.USD.Is(expense.Expense.Currency) {
				usdAmount += expense.Expense.Amount
			} else {
				vedAmount += expense.Expense.Amount
			}
		}
	}

	if vedAmount != 0 {

		amount := (usdAmount * usdRate) + vedAmount

		return AmountWithCurrency{
			Amount:   amount,
			Currency: util.VED,
		}
	}

	return AmountWithCurrency{
		Amount:   usdAmount,
		Currency: util.USD,
	}
}
