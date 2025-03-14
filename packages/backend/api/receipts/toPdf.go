package receipts

import (
	"archive/zip"
	"aws_h"
	"bytes"
	"context"
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"io"
	"kyotaidoshin/util"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

type PdfItem struct {
	ObjectKey string `json:"objectKey"`
	Html      string `json:"html"`
}
type PartInfoUpload struct {
	FileName  string
	ObjectKey string
	Apt       model.Apartments
	Component templ.Component
}

func checkOrBuild(ctx context.Context, parts []PartInfoUpload, isPdf bool) ([]PdfItem, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return nil, err
	}

	if len(parts) == 1 {
		pdfItems := make([]PdfItem, 0)
		part := parts[0]
		exists, err := aws_h.FileExistsS3(ctx, bucketName, part.ObjectKey)
		if err != nil {
			return nil, err
		}

		if exists {
			log.Printf("Skipping %s", part.ObjectKey)
			return pdfItems, err
		}

		var buf bytes.Buffer
		err = part.Component.Render(ctx, &buf)

		if err != nil {
			return nil, err
		}

		base64Str := base64.URLEncoding.EncodeToString(buf.Bytes())

		pdfItems = append(pdfItems, PdfItem{
			ObjectKey: part.ObjectKey,
			Html:      base64Str,
		})
		return pdfItems, err
	}

	numOfWorkers := len(parts)
	var wg sync.WaitGroup
	wg.Add(numOfWorkers)
	itemChan := make(chan PdfItem, numOfWorkers)
	errorChan := make(chan error, numOfWorkers)

	for _, part := range parts {

		go func() {
			defer wg.Done()

			exists, err := aws_h.FileExistsS3(ctx, bucketName, part.ObjectKey)
			if err != nil {
				errorChan <- err
				return
			}

			if exists {
				log.Printf("Skipping %s", part.ObjectKey)
				return
			}

			var buf bytes.Buffer
			err = part.Component.Render(ctx, &buf)

			if err != nil {
				errorChan <- err
				return
			}

			if isPdf {
				base64Str := base64.URLEncoding.EncodeToString(buf.Bytes())
				itemChan <- PdfItem{
					ObjectKey: part.ObjectKey,
					Html:      base64Str,
				}
			} else {
				_, err = aws_h.PutBuffer(ctx, bucketName, part.ObjectKey, "text/html", &buf)

				if err != nil {
					errorChan <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errorChan)
	close(itemChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	pdfItems := make([]PdfItem, 0)

	for item := range itemChan {
		pdfItems = append(pdfItems, item)
	}

	return pdfItems, err
}

func GetParts(receipt *CalculatedReceipt, ctx context.Context, isPdf bool, keys *DownloadKeys) ([]PartInfoUpload, error) {
	functionName, err := resource.Get("HtmlToPdf", "name")
	if err != nil {
		return nil, err
	}

	lambdaClient, err := aws_h.GetLambdaClient(ctx)
	if err != nil {
		return nil, err
	}

	suffix := "pdf"
	if !isPdf {
		suffix = "html"
	}

	buildObjectKey := func(str string) string {
		date := receipt.Receipt.Date.Format(time.DateOnly)

		return fmt.Sprintf("RECEIPTS/%s/%s/%s_%s_%s_%s.%s", receipt.Building.ID, receipt.Receipt.ID,
			receipt.Building.ID, strings.ToUpper(receipt.MonthStr), date, str, suffix)
	}

	parts := make([]PartInfoUpload, 0)
	isOnlyBuilding := keys != nil && len(keys.Parts) == 1 && slices.Contains(keys.Parts, receipt.Building.ID)

	if keys == nil || isOnlyBuilding {
		parts = append(parts, PartInfoUpload{
			FileName:  fmt.Sprintf("%s.%s", receipt.Building.ID, suffix),
			ObjectKey: buildObjectKey(receipt.Building.ID),
			Component: PrintView(receipt.Building.ID, BuildingView(*receipt)),
		})
	}

	if keys == nil || keys.AllApt || (len(keys.Parts) > 0 && !isOnlyBuilding) {
		for _, apt := range receipt.Apartments {
			if keys == nil || keys.AllApt || slices.Contains(keys.Parts, apt.Apartment.Number) {
				parts = append(parts, PartInfoUpload{
					FileName:  fmt.Sprintf("%s.%s", apt.Apartment.Number, suffix),
					ObjectKey: buildObjectKey(apt.Apartment.Number),
					Component: PrintView(apt.Apartment.Number, AptView(*receipt, apt)),
					Apt:       apt.Apartment,
				})
			}
		}

	}

	pdfItems, err := checkOrBuild(ctx, parts, isPdf)
	if err != nil {
		return nil, err
	}

	if len(pdfItems) > 0 {
		jsonBytes, err := json.Marshal(pdfItems)
		if err != nil {
			return nil, err
		}

		_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName:   aws.String(functionName.(string)),
			InvocationType: types.InvocationTypeRequestResponse,
			Payload:        jsonBytes,
		})

		if err != nil {
			return nil, err
		}
	}

	return parts, nil
}

func toZip(receipt *CalculatedReceipt, ctx context.Context, isPdf bool) (*bytes.Buffer, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return nil, err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	parts, err := GetParts(receipt, ctx, isPdf, nil)
	if err != nil {
		return nil, err
	}

	numOfWorkers := len(parts)

	var wg sync.WaitGroup
	wg.Add(numOfWorkers)
	mapByteArray := make(map[int][]byte, numOfWorkers)
	errorChan := make(chan error, numOfWorkers)
	for i, part := range parts {
		go func() {
			defer wg.Done()

			res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(part.ObjectKey),
			})
			if err != nil {
				errorChan <- err
				return
			}

			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(res.Body)

			byteArray, err := io.ReadAll(res.Body)
			if err != nil {
				errorChan <- err
				return
			}

			mapByteArray[i] = byteArray
		}()

	}

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	defer func(zipWriter *zip.Writer) {
		_ = zipWriter.Close()
	}(zipWriter)

	for i, part := range parts {
		writer, err := zipWriter.Create(part.FileName)
		if err != nil {
			return nil, err
		}

		_, err = writer.Write(mapByteArray[i])
		if err != nil {
			return nil, err
		}
	}

	return &buf, nil
}
