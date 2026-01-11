package receipts

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/util"
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
		ids, err = buildings.NewRepository(ctx).SelectIds()
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()

		ratesArray, err := rates.NewRepository(ctx).SelectList(rates.RequestQuery{
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

		fileName = strings.TrimSpace(res.Metadata["filename"])
		if fileName == "" {
			errorChan <- fmt.Errorf("filename not found in metadata")
			return
		}

		filePath := util.TmpFileName(fmt.Sprintf("receipt-%s-%s", util.UuidV7(), fileName))

		file, err := os.Create(filePath)
		if err != nil {
			errorChan <- fmt.Errorf("error creating file %s - %s", filePath, err)
			return
		}

		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		_, err = io.Copy(file, res.Body)
		if err != nil {
			errorChan <- fmt.Errorf("error saving file %s: %s", filePath, err)
			return
		}

		log.Printf("Parsing receipt: %s", fileName)

		if strings.HasSuffix(fileName, ".xlsx") {
			parsedReceipt, err = ExcelizeParser{}.parseWorkbook(filePath)
			//parsedReceipt, err = GrateParser{}.parseWorkbook(filePath)
		} else {
			parsedReceipt, err = ShakinmXlsParser{}.parseWorkbook(filePath)
		}

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

type extraChargeKey struct {
	cell        int
	description string
	amount      float64
	apts        []string
}

type Parser interface {
	parseWorkbook(filePath string) (*ParsedReceipt, error)
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
			newRow = append(newRow, removeSpaces(col))
		}
	}

	return newRow
}

func removeSpaces(str string) string {
	str = strings.Join(strings.Fields(str), " ")
	str = strings.TrimSpace(str)
	return str
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
