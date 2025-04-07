package isr

import (
	"aws_h"
	"bytes"
	"context"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"log"
	"os"
	"strings"
)

func Invoke(ctx context.Context) {

	function, err := resource.Get("IsrGenFunction", "name")
	if err != nil {
		log.Printf("IsrGenFunction not found: %v", err)
		return
	}

	lambdaClient, err := aws_h.GetLambdaClient(ctx)
	if err != nil {
		log.Printf("Failed to get Lambda client: %v", err)
		return
	}

	_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(function.(string)),
		InvocationType: "Event",
	})

	if err != nil {
		log.Printf("Failed to invoke lambda: %s %v", function, err)
		return
	}
}

func putInBucket(ctx context.Context, objectKey string, component templ.Component) error {
	bucket, err := resource.Get("WebAssetsBucket", "name")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = component.Render(ctx, &buf)
	if err != nil {
		return err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return err
	}

	contentLength := int64(buf.Len())
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(bucket.(string)),
		Key:               aws.String(os.Getenv("ISR_PREFIX") + objectKey),
		Body:              &buf,
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		//ChecksumCRC32:             nil,
		//ChecksumCRC32C:            nil,
		//ChecksumSHA1:              nil,
		//ChecksumSHA256:            nil,
		ContentLength: &contentLength,
		ContentType:   aws.String("text/html;charset=UTF-8"),
		CacheControl:  aws.String("public,max-age=0,s-maxage=0,must-revalidate"),
		//CacheControl: aws.String("public,max-age=0,must-revalidate"),
		//CacheControl: aws.String("max-age=0,no-cache,no-store,must-revalidate"), //Works but no 304
	})

	log.Printf("Updated %s %d", objectKey, contentLength)

	if err != nil {
		return err
	}

	return nil
}

func getObject(ctx context.Context, objectKey string, putIfNotExists func(ctx context.Context) error) ([]byte, error) {
	bucket, err := resource.Get("WebAssetsBucket", "name")
	if err != nil {
		return nil, err
	}

	objectKey = os.Getenv("ISR_PREFIX") + objectKey

	exists, err := aws_h.FileExistsS3(ctx, bucket.(string), objectKey)
	if err != nil {
		return nil, err
	}

	if exists {
		data, err := aws_h.GetObject(ctx, bucket.(string), objectKey)
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	err = putIfNotExists(ctx)
	if err != nil {
		return nil, err
	}

	data, err := aws_h.GetObject(ctx, bucket.(string), objectKey)
	if err != nil {
		return nil, err
	}

	return data, nil
}

const ratesCurrenciesObjectKey = "/rates/currencies.html"

func GetRatesCurrencies(ctx context.Context) ([]byte, error) {

	return getObject(ctx, ratesCurrenciesObjectKey, UpdateRatesCurrencies)
}

func UpdateRatesCurrencies(ctx context.Context) error {
	component, err := toStringArray("currencies", getDistinctCurrencies)
	if err != nil {
		return err
	}

	return putInBucket(ctx, ratesCurrenciesObjectKey, component)
}

const receiptsBuildingsObjectKey = "/receipts/buildings.html"

func GetReceiptsBuildings(ctx context.Context) ([]byte, error) {

	return getObject(ctx, receiptsBuildingsObjectKey, updateReceiptsBuildings)
}

func updateReceiptsBuildings(ctx context.Context) error {

	component, err := toStringArray("buildings", receiptBuildings)
	if err != nil {
		return err
	}

	return putInBucket(ctx, receiptsBuildingsObjectKey, component)
}

const receiptsYearsObjectKey = "/receipts/years.html"

func GetReceiptsYears(ctx context.Context) ([]byte, error) {
	return getObject(ctx, receiptsYearsObjectKey, updateReceiptsYears)
}

func updateReceiptsYears(ctx context.Context) error {

	array, err := receiptYears()
	if err != nil {
		return err
	}

	var builder strings.Builder
	builder.WriteString("years = [")
	for i, year := range array {
		builder.WriteString(fmt.Sprint(year))
		if i < len(array)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	return putInBucket(ctx, receiptsYearsObjectKey, XInitView(builder.String()))
}

const receiptApartmentsObjectKey = "/receipts/apartments.html"

func GetReceiptsApartments(ctx context.Context) ([]byte, error) {
	return getObject(ctx, receiptApartmentsObjectKey, updateReceiptsApartments)
}

func updateReceiptsApartments(ctx context.Context) error {
	apts, err := receiptApts()
	if err != nil {
		return err
	}

	return putInBucket(ctx, receiptApartmentsObjectKey, SendAptsView(*apts))
}

const apartmentsBuildingsObjectKey = "/apartments/buildings.html"

func GetApartmentsBuildings(ctx context.Context) ([]byte, error) {
	return getObject(ctx, apartmentsBuildingsObjectKey, updateApartmentsBuildings)
}

func updateApartmentsBuildings(ctx context.Context) error {

	component, err := toStringArray("buildings", apartmentBuildings)
	if err != nil {
		return err
	}

	return putInBucket(ctx, apartmentsBuildingsObjectKey, component)
}

func UpdateAll(ctx context.Context) error {

	functions := []func(ctx context.Context) error{
		UpdateRatesCurrencies,
		updateReceiptsBuildings,
		updateReceiptsYears,
		updateApartmentsBuildings,
		updateReceiptsApartments,
	}

	for _, f := range functions {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func toStringArray(varName string, repoF func() ([]string, error)) (templ.Component, error) {
	response, err := repoF()
	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	builder.WriteString(varName)
	builder.WriteString(" = [")
	for i, currency := range response {
		builder.WriteString(fmt.Sprintf("\"%s\"", currency))
		if i < len(response)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	return XInitView(builder.String()), err
}
