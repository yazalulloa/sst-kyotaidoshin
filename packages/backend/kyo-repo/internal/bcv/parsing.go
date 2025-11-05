package bcv

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/shakinm/xlsReader/xls"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/telegram"
	"github.com/yaz/kyo-repo/internal/util"
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
		processAll, _ = strconv.ParseBool(output.Metadata[MetadataProcessedKey])
	}

	info := ParsingInfo{
		BucketKey:  objKey,
		Data:       data,
		ProcessAll: processAll,
		Ctx:        params.Ctx,
	}

	result, err := info.Parse()
	if err != nil {
		return err
	}

	output.Metadata[MetadataProcessedKey] = "true"
	output.Metadata[MetadataLastProcessedKey] = time.Now().Format(time.RFC3339)
	output.Metadata[MetadataRatesParsedKey] = fmt.Sprint(result.Parsed)
	output.Metadata[MetadataNumOfSheetsKey] = fmt.Sprint(result.NumOfSheets)

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
	return fmt.Sprintf("error parsing %s sheet %s row %d cell %d value [%v]: %v", e.BucketKey, e.SheetName, e.RowIndex, e.CellIndex, e.Value, e.Err)
}

type Result struct {
	Inserted    int64
	Parsed      int
	NumOfSheets int
	FileDate    time.Time
}

type ParsingInfo struct {
	BucketKey  string
	Data       []byte
	FilePath   string
	ProcessAll bool
	Ctx        context.Context
}

func (info ParsingInfo) Parse() (*Result, error) {

	location, err := util.TzCss()
	if err != nil {
		return nil, err
	}

	parsingError := ParsingError{
		BucketKey: info.BucketKey,
	}

	//reader := bytes.NewReader(info.Data)

	result := Result{}
	workbook, err := xls.OpenFile(info.FilePath)

	if err != nil {
		return nil, fmt.Errorf("error opening workbook: %w", err)
	}

	result.NumOfSheets = workbook.GetNumberSheets()

	var rateArray []*model.Rates

	for sheetIndex, sheet := range workbook.GetSheets() {
		parsingError.SheetName = sheet.GetName()
		var dateOfFile *time.Time
		var dateOfRate time.Time
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

					if split[2] == "PM" {
						hour += 12
					}

					minute, err := strconv.Atoi(timeSplit[1])
					if err != nil {
						parsingError.Value = timeSplit[1]
						return nil, parsingError.err(err)
					}

					temp := time.Date(year, time.Month(month), day, hour, minute, 0, 0, location)
					dateOfFile = &temp
				} else {
					fValue := col6.GetFloat64()
					if fValue != 0 {
						log.Printf("Excel date format detected: %s %s %f", info.BucketKey, sheet.GetName(), fValue)
						temp := util.ParseExcelDate(fValue, location)
						dateOfFile = &temp
						log.Printf("Parsed excel dateOfFile: %s", dateOfFile)
					}
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
					log.Printf("ALT Parsed dateOfFile from row 4: %s", dateOfFile)
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
					Rate:         rateF,
					DateOfRate:   dateOfRate,
					DateOfFile:   *dateOfFile,
				}

				rateArray = append(rateArray, &modelRates)
			}

		}

		if dateOfFile == nil {
			return nil, parsingError.err(fmt.Errorf("dateOfFile is nil"))
		}

		result.FileDate = *dateOfFile

		if !info.ProcessAll && sheetIndex == 0 {
			break
		}

		//log.Printf("Sheet: %s rates: %d %s %s", sheet.GetName(), len(rateArray), dateOfRate, dateOfFile)

	}

	result.Parsed += len(rateArray)

	for i := 0; i < len(rateArray); i++ {
		lhs := rateArray[i]
		var rhs *model.Rates
		for j := i + 1; j < len(rateArray); j++ {
			v := rateArray[j]
			if lhs.FromCurrency == v.FromCurrency {
				rhs = v
				break
			}
		}

		diff := 0.00
		diffPercent := 0.00
		trend := rates.STABLE
		if rhs != nil {
			previousRate := lhs.Rate
			nextRate := rhs.Rate

			diff = previousRate - nextRate

			if nextRate != 0 {
				diffPercent = (math.Abs(diff) / nextRate) * 100
				diffPercent = util.RoundFloat(diffPercent, 2)
			}

			if previousRate > nextRate {
				trend = rates.UP
			} else if previousRate < nextRate {
				trend = rates.DOWN
			}
		}

		lhs.ToCurrency = "VED"
		lhs.Source = "BCV"
		lhs.Trend = trend.Name()
		lhs.Diff = diff
		lhs.DiffPercent = diffPercent
	}

	log.Printf("Inserting %d rates from file %s", len(rateArray), info.BucketKey)

	repo := rates.NewRepository(info.Ctx)

	ratesInserted, err := repo.Insert(rateArray)
	result.Inserted += ratesInserted

	//array := util.SplitArray(rateArray, 300)
	//
	//length := len(array)
	//var wg sync.WaitGroup
	//wg.Add(length)
	//errorChan := make(chan error, length)
	//
	//for _, chunk := range array {
	//	go func() {
	//		ratesInserted, err := repo.Insert(chunk)
	//		if err != nil {
	//			errorChan <- fmt.Errorf("error inserting %d rates:  %w", len(rateArray), err)
	//			return
	//		}
	//		result.Inserted += ratesInserted
	//	}()
	//}
	//
	//wg.Wait()
	//close(errorChan)
	//
	//err = util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	//log.Printf("Sheet: %s rates %d inserted: %d", sheet.GetName(), len(rateArray), ratesInserted)

	if !info.ProcessAll && result.Inserted > 0 {
		for _, rate := range rateArray {
			if rate.FromCurrency == "USD" {
				log.Printf("Sending USD rate: %f", rate.Rate)
				telegram.SendRate(info.Ctx, *rate)
			}
		}
	}

	return &result, nil
}
