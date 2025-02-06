package file

import (
	"bcv/bcv"
	"bytes"
	"context"
	"db/gen/model"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/shakinm/xlsReader/xls"
	"io"
	"kyotaidoshin/api"
	"kyotaidoshin/rates"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func ParseFile(ctx context.Context, bucket, key string) error {
	client, err := bcv.GetS3Client()
	if err != nil {
		return err
	}

	objKey := strings.ReplaceAll(key, "%3D", "=")

	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objKey),
	})

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing response body:", err)
			return
		}
	}(output.Body)

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	ratesInserted, err := Parse(data)
	if err != nil {
		return err
	}

	output.Metadata["processed"] = "true"

	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:            &bucket,
		Key:               &objKey,
		CopySource:        aws.String(bucket + "/" + key),
		Metadata:          output.Metadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	})

	log.Printf("Processed %s %s rates %d", key, objKey, ratesInserted)

	if err != nil {
		return fmt.Errorf("error copying object: %w", err)
	}

	return nil
}

func Parse(data []byte) (int64, error) {
	reader := bytes.NewReader(data)

	var inserted int64 = 0
	workbook, err := xls.OpenReader(reader)

	if err != nil {
		return 0, fmt.Errorf("error opening workbook: %w", err)
	}

	for sheetIndex, sheet := range workbook.GetSheets() {

		log.Printf("Sheet %d %s", sheetIndex, sheet.GetName())

		var dateOfFile time.Time
		var dateOfRate time.Time
		var rateArray []model.Rates
		for rowIndex, row := range sheet.GetRows() {

			if rowIndex == 0 {

				col6, err := row.GetCol(6)
				if err != nil {
					return 0, err
				}

				split := strings.Split(col6.GetString(), " ")
				dateSplit := strings.Split(split[0], "/")
				day, err := strconv.Atoi(dateSplit[0])
				if err != nil {
					return 0, err
				}
				month, err := strconv.Atoi(dateSplit[1])
				if err != nil {
					return 0, err
				}

				year, err := strconv.Atoi(dateSplit[2])
				if err != nil {
					return 0, err
				}

				timeSplit := strings.Split(split[1], ":")
				hour, err := strconv.Atoi(timeSplit[0])
				if err != nil {
					return 0, err
				}

				if len(split) > 2 && split[2] == "PM" {
					hour += 12
				}

				minute, err := strconv.Atoi(timeSplit[1])
				if err != nil {
					return 0, err
				}
				location, err := time.LoadLocation("America/Caracas")
				if err != nil {
					return 0, err
				}

				dateOfFile = time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)
			}

			if rowIndex == 4 {
				col3, err := row.GetCol(3)
				if err != nil {
					return 0, err
				}

				cellValue3 := col3.GetString()

				date := strings.TrimSpace(cellValue3[strings.Index(cellValue3, ":")+1:])
				split := strings.Split(date, "/")
				day, err := strconv.Atoi(split[0])
				if err != nil {
					return 0, err
				}
				month, err := strconv.Atoi(split[1])
				if err != nil {
					return 0, err
				}

				year, err := strconv.Atoi(split[2])
				if err != nil {
					return 0, err
				}
				location, err := time.LoadLocation("America/Caracas")
				if err != nil {
					return 0, err
				}

				dateOfRate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
			}

			if rowIndex > 9 {

				col1, err := row.GetCol(1)
				if err != nil {
					return 0, err
				}

				currency := col1.GetString()

				if len(currency) != 3 {
					break
				}

				col6, err := row.GetCol(6)
				if err != nil {
					return 0, err
				}

				rate := strings.ReplaceAll(col6.GetString(), ",", "")
				rateF, err := strconv.ParseFloat(rate, 64)
				if err != nil {
					return 0, err
				}

				str := dateOfRate.Format("20250101") + api.ToASCII(currency) + api.PadLeft(strconv.Itoa(sheetIndex), 4)
				id, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return 0, err
				}

				modelRates := model.Rates{
					ID:           &id,
					FromCurrency: currency,
					ToCurrency:   "VES",
					Rate:         rateF,
					DateOfRate:   dateOfRate,
					Source:       "BCV",
					DateOfFile:   dateOfFile,
					//Hash:         &receiver.Hash,
					//Etag:         receiver.etag,
					//LastModified: receiver.lastModified,
				}

				rateArray = append(rateArray, modelRates)
			}

		}

		//log.Printf("Sheet: %s rates: %d %s %s", sheet.GetName(), len(rateArray), dateOfRate, dateOfFile)
		ratesInserted, err := processRates(&rateArray)
		if err != nil {
			return 0, err
		}

		inserted += ratesInserted
	}

	return inserted, nil
}

func processRates(rateArray *[]model.Rates) (int64, error) {
	ratesToInsert, err := rates.CheckRateInsert(rateArray)
	if err != nil {
		return 0, err
	}

	if len(ratesToInsert) > 0 {
		rowsAffected, err := rates.Insert(ratesToInsert)
		if err != nil {
			return 0, err
		}

		return rowsAffected, nil
	}

	return 0, nil
}

func toASCII(str string) string {
	runes := []rune(str)
	for i, r := range runes {
		if r > unicode.MaxASCII {
			runes[i] = unicode.ReplacementChar
		}
	}
	return string(runes)
}
