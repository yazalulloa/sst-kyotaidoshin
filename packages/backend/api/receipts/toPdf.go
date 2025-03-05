package receipts

import (
	"archive/zip"
	"aws_h"
	"bytes"
	"context"
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
	Component templ.Component
}

func checkOrBuild(ctx context.Context, parts []PartInfoUpload) ([]PdfItem, error) {
	bucketName, err := resource.Get("ReceiptsBucket", "name")
	if err != nil {
		return nil, err
	}

	if len(parts) == 1 {
		pdfItems := make([]PdfItem, 0)
		part := parts[0]
		exists, err := aws_h.FileExistsS3(ctx, bucketName.(string), part.ObjectKey)
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

			exists, err := aws_h.FileExistsS3(ctx, bucketName.(string), part.ObjectKey)
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

			base64Str := base64.URLEncoding.EncodeToString(buf.Bytes())
			itemChan <- PdfItem{
				ObjectKey: part.ObjectKey,
				Html:      base64Str,
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

func getParts(receipt *CalculatedReceipt, ctx context.Context, keys *DownloadKeys) ([]PartInfoUpload, error) {
	functionName, err := resource.Get("HtmlToPdf", "name")
	if err != nil {
		return nil, err
	}

	lambdaClient, err := aws_h.GetLambdaClient(ctx)
	if err != nil {
		return nil, err
	}

	var numOfWorkers int
	if keys == nil {
		numOfWorkers = len(receipt.Apartments) + 1
	} else {
		numOfWorkers = 1
	}

	buildObjectKey := func(str string) string {
		date := receipt.Receipt.Date.Format(time.DateOnly)
		return fmt.Sprintf("%s/%s/%d/%s_%s_%s_%s.pdf", receipt.Building.ID, date, *receipt.Receipt.ID,
			receipt.Building.ID, strings.ToUpper(receipt.MonthStr), date, str)
	}

	parts := make([]PartInfoUpload, numOfWorkers)
	if keys == nil || (!keys.IsApt && keys.Part == receipt.Building.ID) {
		parts[0] = PartInfoUpload{
			FileName:  fmt.Sprintf("%s.pdf", receipt.Building.ID),
			ObjectKey: buildObjectKey(receipt.Building.ID),
			Component: PrintView(receipt.Building.ID, BuildingView(*receipt)),
		}
	}

	if keys == nil || keys.IsApt {
		for i, apt := range receipt.Apartments {
			index := -1
			if keys == nil {
				index = i + 1
			} else {
				if apt.Apartment.Number == keys.Part {
					index = 0
				}
			}

			if index >= 0 {
				parts[index] = PartInfoUpload{
					FileName:  fmt.Sprintf("%s.pdf", apt.Apartment.Number),
					ObjectKey: buildObjectKey(apt.Apartment.Number),
					Component: PrintView(apt.Apartment.Number, AptView(*receipt, apt)),
				}
			}
		}

	}

	pdfItems, err := checkOrBuild(ctx, parts)
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

func toZip(receipt *CalculatedReceipt, ctx context.Context) (*bytes.Buffer, error) {
	bucketName, err := resource.Get("ReceiptsBucket", "name")
	if err != nil {
		return nil, err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	parts, err := getParts(receipt, ctx, nil)
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
				Bucket: aws.String(bucketName.(string)),
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
