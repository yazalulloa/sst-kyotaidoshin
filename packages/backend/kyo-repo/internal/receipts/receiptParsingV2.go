package receipts

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/pbnjay/grate"
	_ "github.com/pbnjay/grate/xls"
	_ "github.com/pbnjay/grate/xlsx"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/debts"
	"github.com/yaz/kyo-repo/internal/expenses"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/util"
)

type GrateParser struct {
}

func (rec GrateParser) parseWorkbook(filePath string) (*ParsedReceipt, error) {
	workbook, err := grate.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer func(workbook grate.Source) {
		err := workbook.Close()
		if err != nil {
			log.Printf("Error closing workbook: %s %v", filePath, err)
		}
	}(workbook)

	sheets, err := workbook.List()
	if err != nil {
		return nil, err
	}

	var expensesArray []model.Expenses
	var debtsArray []model.Debts
	extraChargesArray := make([]model.ExtraCharges, 0)

	for sheetIndex, sheetName := range sheets {

		sheet, err := workbook.Get(sheetName)
		if err != nil {
			return nil, err
		}

		switch sheetIndex {
		case 0:
			array, err := rec.parseExpenses(&sheet)
			if err != nil {
				return nil, err
			}
			expensesArray = array
			continue
		case 1:
			array, err := rec.parseDebts(&sheet)
			if err != nil {
				return nil, err
			}
			debtsArray = array
			continue

		case 2, 4:
			array, err := rec.parseExtraCharges(&sheet)
			if err != nil {
				return nil, err
			}
			extraChargesArray = append(extraChargesArray, array...)
		}

	}

	dest := ParsedReceipt{
		Expenses:     expensesArray,
		Debts:        debtsArray,
		ExtraCharges: extraChargesArray,
	}

	return &dest, nil

}

func (rec GrateParser) parseExpenses(sheetP *grate.Collection) ([]model.Expenses, error) {
	array := make([]model.Expenses, 0)
	expenseType := expenses.COMMON

	sheet := *sheetP

	rowIndex := 0
	for sheet.Next() {
		row := trimRow(sheet.Strings())

		if len(row) > 0 {
			description := row[0]

			if strings.Contains(description, "GASTOS NO COMUNES") || strings.Contains(description, "TOTAL GASTOS COMUNES") {
				expenseType = expenses.UNCOMMON
				continue
			}

			if strings.Contains(description, "TOTAL GASTOS NO COMUNES") ||
				strings.Contains(description, "TOTAL GASTOS DEL MES") ||
				strings.Contains(description, "TOTAL GASTOS DE MES") {
				break
			}

			if strings.Contains(description, "FONDOS DE") || strings.Contains(description, "TOTAL GASTOS DEL MES") {
				break
			}

			if len(row) >= 2 {
				amountStr := row[1]

				amountStr, err := toAmount(amountStr)
				if err != nil {
					return nil, fmt.Errorf("expenses -> row %d: %s", rowIndex, err)
				}

				amount, err := strconv.ParseFloat(amountStr, 64)
				if err != nil {
					return nil, fmt.Errorf("expenses failed to parse amount -> row %d: %s", rowIndex, err)
				}

				expense := model.Expenses{
					Description: description,
					Amount:      amount,
					Type:        expenseType.Name(),
					Currency:    "VED",
				}

				array = append(array, expense)
			}

		}

		rowIndex++
	}

	return array, nil
}

func (rec GrateParser) parseDebts(sheetP *grate.Collection) ([]model.Debts, error) {
	array := make([]model.Debts, 0)

	sheet := *sheetP

	rowIndex := 0
	for sheet.Next() {
		row := trimRow(sheet.Strings())

		if len(row) >= 4 {
			apt := row[0]

			if apt == "APTO" || apt == "MONTOS EN DOLARES" {
				continue
			}

			if apt == "" {
				continue
			}

			receipts := util.StringToInt16(strings.TrimSpace(strings.Replace(row[2], "CUOTA", "", -1)))
			amountStr := row[3]

			if len(amountStr) == 0 || amountStr == "MONTO" {
				continue
			}

			amountStr, err := toAmount(amountStr)
			if err != nil {
				return nil, fmt.Errorf("debts -> row %d: %s", rowIndex, err)
			}

			amount := 0.00

			if amountStr != "" {
				amount, err = strconv.ParseFloat(amountStr, 64)
				if err != nil {
					return nil, fmt.Errorf("debts failed to parse amount -> row %d: %s", rowIndex, err)
				}
			}

			status := ""

			if len(row) > 4 {
				status = row[4]
			}

			abono := ""

			if len(row) > 5 {
				abono = row[5]
			}

			if abono == "" && hasDigits(status) {
				abono = status
			}

			previousPaymentAmount := 0.00
			previousPaymentCurrency := "VED"

			if strings.Contains(abono, "$") {
				previousPaymentCurrency = "USD"
			}

			if abono != "" {

				abono = stringReplaceArray(abono, "$", "Bs.")
				if abono != "" && abono != "OJO" && abono != "ABONO" && !strings.Contains(abono, "MESES") {
					abono, err = toAmount(abono)
					if err != nil {
						return nil, fmt.Errorf("debts -> row %d: %s", rowIndex, err)
					}

					previousPaymentAmount, err = strconv.ParseFloat(abono, 64)
					if err != nil {
						log.Printf("debts failed to parse previous payment amount -> row %d: %s", rowIndex, err)
					}
				}
			}

			years := make([]debts.YearWithMonths, 0)
			monthlyDebt := debts.MonthlyDebt{
				Amount: 0,
			}

			if strings.Contains(status, "MESES") {

				str := util.RemoveNonNumeric(status)
				amount, err := strconv.Atoi(str)
				if err != nil {
					return nil, err
				}

				monthlyDebt.Amount = amount

			} else {
				split := strings.Split(status, "/")
				months := make([]int16, 0)
				for _, s := range split {
					if s != "" {
						month := util.MonthToInt16(s)
						if month != 0 {
							months = append(months, month)
						}
					}
				}

				if len(months) > 0 {
					years = append(years, debts.YearWithMonths{
						Year:   0,
						Months: months,
					})
				}
			}

			monthlyDebt.Years = years

			byteArray, err := json.Marshal(monthlyDebt)
			if err != nil {
				return nil, err
			}

			apt = strings.TrimSuffix(apt, ".")

			debt := model.Debts{
				AptNumber:                     apt,
				Receipts:                      receipts,
				Amount:                        amount,
				Months:                        string(byteArray),
				PreviousPaymentAmount:         previousPaymentAmount,
				PreviousPaymentAmountCurrency: previousPaymentCurrency,
			}

			array = append(array, debt)
		}

		rowIndex++
	}

	return array, nil
}

func (rec GrateParser) parseExtraCharges(sheetP *grate.Collection) ([]model.ExtraCharges, error) {
	array := make([]extraChargeKey, 0)

	checkIfHasDesc := func(array []string) bool {
		for _, s := range array {
			if s == "APTO" || s == "MONTO" {
				return true
			}
		}
		return false
	}

	afterDescriptions := false
	sheet := *sheetP

	rowIndex := 0
	for sheet.Next() {
		row := trimRow(sheet.Strings())

		if len(row) > 0 {

			ifHasDesc := checkIfHasDesc(row)
			if ifHasDesc {
				afterDescriptions = true
				continue
			}

			if !afterDescriptions {
				for cellIndex, cell := range row {

					cell = strings.TrimSpace(cell)

					if cell == "" || strings.Contains(cell, "PARA SER CARGADOS INDIVIDUALMENTE A CADA APTO") {
						continue
					}

					array = append(array, extraChargeKey{
						cell:        cellIndex,
						description: cell,
					})
				}

				continue
			}

			previousApt := ""
			for cellIndex, cell := range row {

				if cell == "" {
					continue
				}

				if previousApt != "" {
					amountStr, err := toAmount(cell)
					if err != nil {
						return nil, fmt.Errorf("extra charges -> row %d cell %d: %s | %s", rowIndex, cellIndex, cell, err)
					}

					amount, err := strconv.ParseFloat(amountStr, 64)
					if err != nil {
						previousApt = ""
						continue
						//return nil, fmt.Errorf("extra charges failed to parse amount -> row %d cell %d: %s | %s", rowIndex, cellIndex, cell, err)
					}

					i := slices.IndexFunc(array, func(key extraChargeKey) bool {
						return key.cell == cellIndex-1 && (key.amount == 0 || key.amount == amount)
					})

					//&& (key.amount == 0 || key.amount == amount)
					if i == -1 {
						i = slices.IndexFunc(array, func(key extraChargeKey) bool {
							return key.cell == cellIndex-1
						})

						if i == -1 {
							log.Printf("this should not happen")
							continue
						}
					}

					chargeKey := array[i]

					if chargeKey.amount == 0 || chargeKey.amount == amount {
						chargeKey.amount = amount
						chargeKey.apts = append(chargeKey.apts, previousApt)
						array[i] = chargeKey
					} else {
						array = append(array, extraChargeKey{
							cell:        cellIndex - 1,
							description: chargeKey.description,
							amount:      amount,
							apts:        []string{previousApt},
						})
					}

					previousApt = ""
					continue
				}

				previousApt = strings.ReplaceAll(cell, "--", "-")
			}
		}

		rowIndex++
	}

	charges := make([]model.ExtraCharges, 0)
	for _, v := range array {

		if v.amount == 0 {
			continue
		}

		charges = append(charges, model.ExtraCharges{
			Type:        extraCharges.TypeReceipt,
			Description: v.description,
			Amount:      v.amount,
			Currency:    util.VED.Name(),
			Active:      true,
			Apartments:  strings.Join(v.apts, ","),
		})
	}

	slices.SortFunc(charges, func(a, b model.ExtraCharges) int {
		return cmp.Or(
			cmp.Compare(a.Description, b.Description),
			cmp.Compare(a.Amount, b.Amount),
		)
	})

	return charges, nil
}
