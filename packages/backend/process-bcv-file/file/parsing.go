package file

import (
	"aws_h"
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
	"kyotaidoshin/rates"
	"kyotaidoshin/util"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type ParsingParams struct {
	Ctx         context.Context
	Bucket, Key string
	ProcessAll  *bool
}

func fileParse(params ParsingParams) error {
	client, err := aws_h.GetS3Client(params.Ctx)
	if err != nil {
		return err
	}

	objKey := strings.ReplaceAll(params.Key, "%3D", "=")

	log.Printf("Processing %s\n", objKey)

	output, err := client.GetObject(params.Ctx, &s3.GetObjectInput{
		Bucket: aws.String(params.Bucket),
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

	var processAll = false
	if params.ProcessAll != nil {
		processAll = *params.ProcessAll
	} else {
		processAll, _ = strconv.ParseBool(output.Metadata[bcv.MetadataProcessedKey])
	}

	info := ParsingInfo{
		BucketKey:  objKey,
		Data:       data,
		ProcessAll: processAll,
	}

	result, err := info.parse()
	if err != nil {
		return err
	}

	output.Metadata[bcv.MetadataProcessedKey] = "true"
	output.Metadata[bcv.MetadataLastProcessedKey] = time.Now().Format(time.RFC3339)
	output.Metadata[bcv.MetadataRatesParsedKey] = strconv.FormatInt(result.Parsed, 10)

	_, err = client.CopyObject(params.Ctx, &s3.CopyObjectInput{
		Bucket:            &params.Bucket,
		Key:               &objKey,
		CopySource:        aws.String(params.Bucket + "/" + params.Key),
		Metadata:          output.Metadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	})

	log.Printf("Processed %s rates parsed: %d inserted: %d", objKey, result.Parsed, result.Inserted)

	if err != nil {
		return fmt.Errorf("error copying object: %w", err)
	}

	return nil
}

func ParseFile(params ParsingParams) error {

	err := fileParse(params)

	if err != nil {
		log.Printf("Error parsing file: %s %v\n", params.Key, err)
	}

	return err
}

type ParsingError struct {
	BucketKey string
	SheetName string
	RowIndex  int
	CellIndex int
	Value     any
	Err       error
}

func (e ParsingError) err(err error) ParsingError {
	e.Err = err
	return e
}

func (e ParsingError) Error() string {
	return fmt.Sprintf("error parsing bucket %s sheet %s row %d cell %d value [%v]: %v", e.BucketKey, e.SheetName, e.RowIndex, e.CellIndex, e.Value, e.Err)
}

type Result struct {
	Inserted int64
	Parsed   int64
}

type ParsingInfo struct {
	BucketKey  string
	Data       []byte
	ProcessAll bool
}

func (info ParsingInfo) parse() (*Result, error) {

	location, err := time.LoadLocation("America/Caracas")
	if err != nil {
		return nil, err
	}

	parsingError := ParsingError{
		BucketKey: info.BucketKey,
	}

	reader := bytes.NewReader(info.Data)

	result := Result{}
	workbook, err := xls.OpenReader(reader)

	if err != nil {
		return nil, fmt.Errorf("error opening workbook: %w", err)
	}

	for sheetIndex, sheet := range workbook.GetSheets() {
		parsingError.SheetName = sheet.GetName()
		var dateOfFile *time.Time
		var dateOfRate time.Time
		var rateArray []model.Rates
		for rowIndex, row := range sheet.GetRows() {
			parsingError.RowIndex = rowIndex

			if rowIndex == 0 {

				col6, err := row.GetCol(6)
				parsingError.CellIndex = 6
				if err != nil {
					nE := parsingError.err(err)
					log.Printf("Error %s", nE.Error())
					return nil, nE
				}

				cellValue := col6.GetString()
				parsingError.Value = cellValue
				split := strings.Split(cellValue, " ")

				if len(split) == 3 {
					dateSplit := strings.Split(split[0], "/")
					day, err := strconv.Atoi(dateSplit[0])
					if err != nil {
						return nil, parsingError.err(err)
					}
					month, err := strconv.Atoi(dateSplit[1])
					if err != nil {
						return nil, parsingError.err(err)
					}

					year, err := strconv.Atoi(dateSplit[2])
					if err != nil {
						return nil, parsingError.err(err)
					}

					timeSplit := strings.Split(split[1], ":")
					hour, err := strconv.Atoi(timeSplit[0])
					if err != nil {
						return nil, parsingError.err(err)
					}

					if len(split) > 2 && split[2] == "PM" {
						hour += 12
					}

					minute, err := strconv.Atoi(timeSplit[1])
					if err != nil {
						parsingError.Value = timeSplit[1]
						return nil, parsingError.err(err)
					}

					temp := time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)
					dateOfFile = &temp
				}

			}

			if rowIndex == 4 {

				if dateOfFile == nil {
					col1, err := row.GetCol(1)
					parsingError.CellIndex = 1
					if err != nil {
						return nil, parsingError.err(err)
					}

					cellValue1 := col1.GetString()
					parsingError.Value = cellValue1

					date := strings.TrimSpace(cellValue1[strings.Index(cellValue1, ":")+1:])
					split := strings.Split(date, "/")
					day, err := strconv.Atoi(split[0])
					if err != nil {
						return nil, parsingError.err(err)
					}

					month, err := strconv.Atoi(split[1])
					if err != nil {
						return nil, parsingError.err(err)
					}

					year, err := strconv.Atoi(split[2])
					if err != nil {
						return nil, parsingError.err(err)
					}

					temp := time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
					dateOfFile = &temp
				}

				col3, err := row.GetCol(3)
				parsingError.CellIndex = 3
				if err != nil {
					return nil, parsingError.err(err)
				}

				cellValue3 := col3.GetString()
				parsingError.Value = cellValue3

				date := strings.TrimSpace(cellValue3[strings.Index(cellValue3, ":")+1:])
				split := strings.Split(date, "/")
				day, err := strconv.Atoi(split[0])
				if err != nil {
					return nil, parsingError.err(err)
				}
				month, err := strconv.Atoi(split[1])
				if err != nil {
					return nil, parsingError.err(err)
				}

				year, err := strconv.Atoi(split[2])
				if err != nil {
					return nil, parsingError.err(err)
				}

				dateOfRate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
			}

			if rowIndex > 9 {

				if dateOfFile == nil {
					log.Printf("Error dateOfFile is nil")
					continue
				}

				col1, err := row.GetCol(1)
				parsingError.CellIndex = 1
				if err != nil {
					return nil, parsingError.err(err)
				}

				currency := col1.GetString()

				if len(currency) != 3 {
					break
				}

				col6, err := row.GetCol(6)
				parsingError.CellIndex = 6
				if err != nil {
					return nil, parsingError.err(err)
				}

				cell6Value := col6.GetString()
				parsingError.Value = cell6Value
				rate := strings.ReplaceAll(cell6Value, ",", "")
				rateF, err := strconv.ParseFloat(rate, 64)
				if err != nil {
					return nil, parsingError.err(err)
				}

				str := strings.ReplaceAll(dateOfRate.Format(time.DateOnly), "-", "") + util.ToASCII(currency)
				id, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					parsingError.Value = str
					return nil, parsingError.err(err)
				}

				modelRates := model.Rates{
					ID:           &id,
					FromCurrency: currency,
					ToCurrency:   "VES",
					Rate:         rateF,
					DateOfRate:   dateOfRate,
					Source:       "BCV",
					DateOfFile:   *dateOfFile,
					//Hash:         &receiver.Hash,
					//Etag:         receiver.etag,
					//LastModified: receiver.lastModified,
				}

				rateArray = append(rateArray, modelRates)
			}

		}

		//log.Printf("Sheet: %s rates: %d %s %s", sheet.GetName(), len(rateArray), dateOfRate, dateOfFile)
		result.Parsed += int64(len(rateArray))

		if info.ProcessAll || sheetIndex == 0 {

			ratesInserted, err := processRates(&rateArray)
			parsingError.Value = ""
			if err != nil {
				return nil, parsingError.err(err)
			}
			result.Inserted += ratesInserted
		}
	}

	return &result, nil
}

func processRates(rateArray *[]model.Rates) (int64, error) {

	rowsAffected, err := rates.Insert(*rateArray)
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
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
