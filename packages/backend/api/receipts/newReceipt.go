package receipts

import (
	"aws_h"
	"bytes"
	"cmp"
	"compress/flate"
	"context"
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"
	"io"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/util"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

func parseNewReceipt(ctx context.Context, key string) (*ReceiptFileFormDto, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return nil, err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	var ids []string
	var rateArray []rates.Option
	var fileName string
	var parsedReceipt *ParsedReceipt

	go func() {
		defer wg.Done()
		ids, err = buildings.SelectIds()
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()

		ratesArray, err := rates.SelectList(rates.RequestQuery{
			Currencies: []string{util.USD.Name()},
			SortOrder:  util.SortOrderTypeDESC,
			Limit:      10,
		})

		if err != nil {
			errorChan <- err
			return
		}

		rateArray = make([]rates.Option, len(ratesArray))
		for i, r := range ratesArray {
			rateArray[i] = rates.Option{
				Key:        *util.Encode(*r.ID),
				DateOfRate: r.DateOfRate.Format(time.DateOnly),
				Rate:       r.Rate,
			}
		}
	}()

	go func() {
		defer wg.Done()

		res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})

		if err != nil {
			errorChan <- err
			return
		}

		defer func() {

			_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
			})

			if err != nil {
				log.Printf("Error deleting object: %s %s %s", bucketName, key, err)
			}
		}()

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		data, err := io.ReadAll(res.Body)
		if err != nil {
			errorChan <- err
			return
		}

		fileName = strings.TrimSpace(res.Metadata["filename"])
		if fileName == "" {
			errorChan <- fmt.Errorf("filename not found in metadata")
			return
		}

		log.Printf("Parsing receipt: %s", fileName)

		reader := bytes.NewReader(data)
		parsedReceipt, err = parseWorkbook(reader)
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

	location, err := time.LoadLocation("America/Caracas")
	if err != nil {
		return nil, err
	}

	now := time.Now().In(location)

	buildingMatched := ids[0]

	for _, s := range ids {
		if strings.Contains(fileName, s) {
			buildingMatched = s
			break
		}
	}

	month := util.GetMonthIfContains(fileName)
	if month == 0 {
		month = int16(now.Month())
	}

	year := int16(now.Year())

	if month == int16(time.December) && now.Month() == time.January {
		year--
	}

	years := []int16{year + 1, year, year - 1, year - 2, year - 3, year - 4}

	byteArray, err := json.Marshal(*parsedReceipt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(byteArray)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	data := base64.URLEncoding.EncodeToString(buf.Bytes())

	return &ReceiptFileFormDto{
		Month:     month,
		Year:      year,
		Years:     years,
		Building:  buildingMatched,
		Buildings: ids,
		Filename:  fileName,
		Date:      now.Format(time.DateOnly),
		Rates:     rateArray,
		Data:      data,
	}, nil
}

type ParsedReceipt struct {
	Expenses     []model.Expenses     `json:"expenses"`
	Debts        []model.Debts        `json:"debts"`
	ExtraCharges []model.ExtraCharges `json:"extra_charges"`
}

func parseWorkbook(reader io.ReadSeeker) (*ParsedReceipt, error) {
	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, err
	}

	var expensesArray []model.Expenses
	var debtsArray []model.Debts
	extraChargesArray := make([]model.ExtraCharges, 0)

	for sheetIndex := 0; sheetIndex < workbook.SheetCount; sheetIndex++ {
		sheetName := workbook.GetSheetName(sheetIndex)
		rows, err := workbook.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
		switch sheetIndex {
		case 0:
			array, err := parseExpenses(&rows)
			if err != nil {
				return nil, err
			}
			expensesArray = array
			continue
		case 1:
			array, err := parseDebts(&rows)
			if err != nil {
				return nil, err
			}
			debtsArray = array
			continue

		case 2, 4:
			array, err := parseExtraCharges(&rows)
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

func parseExpenses(rows *[][]string) ([]model.Expenses, error) {
	array := make([]model.Expenses, 0)
	expenseType := expenses.COMMON
	for rowIndex, row := range *rows {

		row = trimRow(row)

		if expenses.UNCOMMON == expenseType && len(row) == 0 {
			break
		}

		if len(row) > 0 {
			description := row[0]

			if strings.Contains(description, "GASTOS NO COMUNES") || strings.Contains(description, "TOTAL GASTOS COMUNES") {
				expenseType = expenses.UNCOMMON
				continue
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
	}

	return array, nil
}

func parseDebts(rows *[][]string) ([]model.Debts, error) {
	array := make([]model.Debts, 0)

	for rowIndex, row := range *rows {

		row = trimRow(row)

		if len(row) >= 4 {
			apt := row[0]

			if apt == "APTO" {
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
				//
				//location, err := time.LoadLocation("America/Caracas")
				//if err != nil {
				//	return nil, err
				//}
				//
				//now := time.Now().In(location)

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

	}

	return array, nil
}

func parseExtraCharges(rows *[][]string) ([]model.ExtraCharges, error) {
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
	for rowIndex, row := range *rows {

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

type extraChargeKey struct {
	cell        int
	description string
	amount      float64
	apts        []string
}

func toAmount(str string) (string, error) {
	point := strings.LastIndex(str, ".")
	comma := strings.LastIndex(str, ",")

	if point == -1 && comma == -1 {
		return str, nil
	}

	if point == -1 {
		count := strings.Count(str, ",")
		if count == 1 {
			return strings.Replace(str, ",", ".", 1), nil
		}

		return strings.Replace(str, ",", "", -1), nil
	}

	if comma == -1 {
		count := strings.Count(str, ".")
		if count == 1 {
			return str, nil
		}

		if point == len(str)-3 {
			return "", fmt.Errorf("invalid amount: %s", str)
		}

		return strings.Replace(str, ".", "", -1), nil
	}

	if point > comma {
		return strings.Replace(str, ",", "", -1), nil
	}

	return strings.Replace(strings.Replace(str, ".", "", -1), ",", ".", -1), nil
}

func trimRow(row []string) []string {
	newRow := make([]string, 0)

	for _, col := range row {
		col = strings.TrimSpace(col)
		if len(col) > 0 {
			col = strings.Join(strings.Fields(col), " ")
			col = strings.TrimSpace(col)
			newRow = append(newRow, col)
		}
	}

	return newRow
}

func stringReplaceArray(str string, array ...string) string {
	for _, s := range array {
		str = strings.Replace(str, s, ",", -1)
	}

	return strings.TrimSpace(str)
}

func hasDigits(str string) bool {
	for _, r := range str {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false

}
