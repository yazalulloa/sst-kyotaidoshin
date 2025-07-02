package isr

import (
	"bytes"
	"context"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"log"
	"os"
	"strings"
)

const ratesCurrenciesObjectKey = "/rates/currencies.html"
const receiptsBuildingsObjectKey = "/receipts/buildings.html"
const receiptsYearsObjectKey = "/receipts/years.html"
const receiptApartmentsObjectKey = "/receipts/apartments.html"
const apartmentsBuildingsObjectKey = "/apartments/buildings.html"

type IsrObj struct {
	objectKey      string
	buildComponent func(context.Context) (templ.Component, error)
}

var isrObjects = []IsrObj{
	{
		objectKey: ratesCurrenciesObjectKey,
		buildComponent: func(ctx context.Context) (templ.Component, error) {
			return toStringArray("currencies", NewRepository(ctx).getDistinctCurrencies)
		},
	},
	{
		objectKey: receiptsBuildingsObjectKey,
		buildComponent: func(ctx context.Context) (templ.Component, error) {
			return toStringArray("buildings", NewRepository(ctx).receiptBuildings)
		},
	},
	{
		objectKey: receiptsYearsObjectKey,
		buildComponent: func(ctx context.Context) (templ.Component, error) {
			array, err := NewRepository(ctx).receiptYears()
			if err != nil {
				return nil, err
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

			return XInitView(builder.String()), nil
		},
	},
	{
		objectKey: receiptApartmentsObjectKey,
		buildComponent: func(ctx context.Context) (templ.Component, error) {
			apts, err := NewRepository(ctx).receiptApts()
			if err != nil {
				return nil, err
			}

			return SendAptsView(*apts), nil

		},
	},
	{
		objectKey: apartmentsBuildingsObjectKey,
		buildComponent: func(ctx context.Context) (templ.Component, error) {
			return toStringArray("buildings", NewRepository(ctx).apartmentBuildings)
		},
	},
}

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

func (isr IsrObj) putInBucket(ctx context.Context) error {
	bucket, err := resource.Get("WebAssetsBucket", "name")
	if err != nil {
		return err
	}

	component, err := isr.buildComponent(ctx)
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
		Key:               aws.String(os.Getenv("ISR_PREFIX") + isr.objectKey),
		Body:              &buf,
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		//ChecksumCRC32:             nil,
		//ChecksumCRC32C:            nil,
		//ChecksumSHA1:              nil,
		//ChecksumSHA256:            nil,
		ContentLength: &contentLength,
		ContentType:   aws.String("text/html;charset=UTF-8"),
		CacheControl:  aws.String("public,max-age=2,s-maxage=2,must-revalidate"),
		//CacheControl:  aws.String("s-maxage=2,stale-while-revalidate=2592000"), // works but it takes at least one request to update, maybe 2 or 3
		//CacheControl:  aws.String("public,max-age=0,s-maxage=0,must-revalidate"),
		//CacheControl: aws.String("public,max-age=0,must-revalidate"),
		//CacheControl: aws.String("max-age=0,no-cache,no-store,must-revalidate"), //Works but no 304
	})

	log.Printf("Updated %s %d", isr.objectKey, contentLength)

	if err != nil {
		return err
	}

	return nil
}

func (isr IsrObj) getObject(ctx context.Context) ([]byte, error) {
	bucket, err := resource.Get("WebAssetsBucket", "name")
	if err != nil {
		return nil, err
	}

	objectKey := os.Getenv("ISR_PREFIX") + isr.objectKey

	exists, err := aws_h.FileExistsS3(ctx, bucket.(string), objectKey)
	if err != nil {
		return nil, err
	}

	if exists {
		data, err := aws_h.GetObjectBuffer(ctx, bucket.(string), objectKey)
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	err = isr.putInBucket(ctx)
	if err != nil {
		return nil, err
	}

	data, err := aws_h.GetObjectBuffer(ctx, bucket.(string), objectKey)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func UpdateAll(ctx context.Context) error {

	for _, obj := range isrObjects {
		err := obj.putInBucket(ctx)
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
