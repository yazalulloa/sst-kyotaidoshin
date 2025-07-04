package receipts

import (
	"archive/zip"
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
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"io"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

type PdfItem struct {
	ObjectKey    string `json:"objectKey"`
	Html         string `json:"html"`
	PresignedUrl string `json:"presignedUrl"`
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
			//log.Printf("Skipping %s", part.ObjectKey)
			return pdfItems, err
		}

		var buf bytes.Buffer
		err = part.Component.Render(ctx, &buf)

		if err != nil {
			return nil, err
		}

		content := buf.String()

		newStr := strings.ReplaceAll(content, "<br>", "<br></br>")
		newStr = strings.Replace(newStr, "!doctype", "!DOCTYPE", 1)
		buf.Reset()

		_, err = buf.WriteString(newStr)
		if err != nil {
			return nil, err
		}

		base64Str := base64.URLEncoding.EncodeToString(buf.Bytes())

		url, err := aws_h.PresignPut(ctx, bucketName, part.ObjectKey, "application/pdf")
		if err != nil {
			return nil, err
		}

		pdfItems = append(pdfItems, PdfItem{
			ObjectKey:    part.ObjectKey,
			Html:         base64Str,
			PresignedUrl: url,
		})
		return pdfItems, err
	}

	numOfWorkers := len(parts)
	var wg sync.WaitGroup
	wg.Add(numOfWorkers)
	itemChan := make(chan PdfItem, numOfWorkers)
	errorChan := make(chan error, numOfWorkers)

	for i, part := range parts {

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

			content := buf.String()

			if i == 2 {
				//log.Printf("Content %d", len(content))
				//log.Printf("Content %s", content)
				//log.Printf("Content %d", len(content))
			}

			newStr := strings.ReplaceAll(content, "<br>", "<br></br>")
			newStr = strings.Replace(newStr, "!doctype", "!DOCTYPE", 1)
			buf.Reset()

			_, err = buf.WriteString(newStr)
			if err != nil {
				errorChan <- err
			}

			if isPdf {
				base64Str := base64.URLEncoding.EncodeToString(buf.Bytes())
				url, err := aws_h.PresignPut(ctx, bucketName, part.ObjectKey, "application/pdf")
				if err != nil {
					errorChan <- err
				}

				itemChan <- PdfItem{
					ObjectKey:    part.ObjectKey,
					Html:         base64Str,
					PresignedUrl: url,
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

func GetElemObjectKey(buildingId, receiptId, date, elem, suffix string, year, month int16) string {
	if elem != "" {
		elem = fmt.Sprintf("_%s", elem)
	}

	objectKey := fmt.Sprintf("RECEIPTS/%s/%d/%s/%s_%d_%s_%s%s.%s", buildingId, year, receiptId,
		buildingId, year, strings.ToUpper(util.FromInt16ToMonth(month)), date, elem, suffix)
	return objectKey
}

func GetParts(receipt *CalculatedReceipt, ctx context.Context, isPdf bool, keys *DownloadKeys) ([]PartInfoUpload, error) {
	//functionName, err := resource.Get("HtmlToPdf", "name")
	//if err != nil {
	//	return nil, err
	//}

	lambdaClient, err := aws_h.GetLambdaClient(ctx)
	if err != nil {
		return nil, err
	}

	suffix := "pdf"
	if !isPdf {
		suffix = "html"
	}

	date := receipt.Receipt.Date.Format(time.DateOnly)

	buildObjectKey := func(str string) string {

		return GetElemObjectKey(receipt.Building.ID, receipt.Receipt.ID, date, str, suffix, receipt.Receipt.Year, receipt.Receipt.Month)
		//return fmt.Sprintf("RECEIPTS/%s/%s/%s_%s_%s_%s.%s", receipt.Building.ID, receipt.Receipt.ID,
		//	receipt.Building.ID, strings.ToUpper(receipt.MonthStr), date, str, suffix)
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
		functionName, err := resource.Get("HtmlToPdfFunction", "value")
		if err != nil {
			return nil, fmt.Errorf("error getting HtmlToPdfFunction: %w", err)
		}

		jsonBytes, err := json.Marshal(pdfItems)
		if err != nil {
			return nil, err
		}

		res, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName:   aws.String(functionName.(string)),
			InvocationType: types.InvocationTypeRequestResponse,
			Payload:        jsonBytes,
		})

		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			return nil, fmt.Errorf("error invoking lambda %d %s", res.StatusCode, string(res.Payload))
		}

		if res.FunctionError != nil {
			return nil, fmt.Errorf("error invoking lambda %s", *res.FunctionError)
		}
	}

	return parts, nil
}

func toZip(receipt *CalculatedReceipt, ctx context.Context, isPdf bool) (string, error) {
	//func toZip(receipt *CalculatedReceipt, ctx context.Context, isPdf bool) (*bytes.Buffer, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return "", err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return "", err
	}

	parts, err := GetParts(receipt, ctx, isPdf, nil)
	if err != nil {
		return "", fmt.Errorf("error getting parts for receipt: %w", err)
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

				errorChan <- fmt.Errorf("error getting object %s from bucket %s: %w", part.ObjectKey, bucketName, err)
				return
			}

			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(res.Body)

			byteArray, err := io.ReadAll(res.Body)
			if err != nil {
				errorChan <- fmt.Errorf("error reading object %s from bucket %s: %w", part.ObjectKey, bucketName, err)
				return
			}

			mapByteArray[i] = byteArray
		}()

	}

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return "", err
	}

	fileName := util.TmpFileName(fmt.Sprintf("receipt_%s_%s_%s.zip", receipt.Building.ID, receipt.Receipt.ID, util.UuidV7()))
	archive, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("error creating archive file: %w", err)
	}

	defer func(archive *os.File) {
		err := archive.Close()
		if err != nil {
			log.Printf("error closing archive file: %v", err)
		}
	}(archive)

	zipWriter := zip.NewWriter(archive)

	defer func(zipWriter *zip.Writer) {
		_ = zipWriter.Close()
	}(zipWriter)

	for i, part := range parts {
		writer, err := zipWriter.Create(part.FileName)
		if err != nil {
			return "", fmt.Errorf("error creating zip writer for %s: %w", part.FileName, err)
		}

		_, err = writer.Write(mapByteArray[i])
		if err != nil {
			return "", fmt.Errorf("error writing to zip for %s: %w", part.FileName, err)
		}
	}

	return fileName, nil
}

type ReceiptZipInfo struct {
	BucketName string
	ObjectKey  string
	FilePath   string
}

func GetZipObjectKey(ctx context.Context, buildingId, receiptId string) (*ReceiptZipInfo, error) {

	bucketName, err := util.GetReceiptsBucket()

	if err != nil {
		return nil, fmt.Errorf("error getting bucket name: %w", err)
	}

	rec, err := NewRepository(ctx).SelectById(receiptId)
	if err != nil {
		return nil, fmt.Errorf("error getting receipt from db: %w", err)
	}

	date := rec.Date.Format(time.DateOnly)
	//objectKey := fmt.Sprintf("RECEIPTS/%s/%s/%s_%d_%s_%s.zip", rec.BuildingID, rec.ID,
	//	rec.BuildingID, rec.Year, util.FromInt16ToMonth(rec.Month), date)

	objectKey := GetElemObjectKey(rec.BuildingID, rec.ID, date, "", "zip", rec.Year, rec.Month)

	exists, err := aws_h.FileExistsS3(ctx, bucketName, objectKey)
	if err != nil {
		return nil, fmt.Errorf("error checking if file exists: %w", err)
	}

	zipInfo := &ReceiptZipInfo{
		BucketName: bucketName,
		ObjectKey:  objectKey,
	}

	if !exists {

		receipt, err := CalculateReceipt(ctx, buildingId, receiptId)
		if err != nil {
			return nil, fmt.Errorf("error calculating receipt: %w", err)
		}

		filePath, err := toZip(receipt, ctx, true)

		if err != nil {
			return nil, fmt.Errorf("error creating zip: %w", err)
		}

		_, err = aws_h.PutFile(ctx, bucketName, objectKey, "application/zip", filePath)

		if err != nil {
			return nil, fmt.Errorf("error uploading zip: %w", err)
		}

		zipInfo.FilePath = filePath
	}

	return zipInfo, nil
}

type ReceiptPdfInfo struct {
	BucketName string
	ObjectKey  string
}
