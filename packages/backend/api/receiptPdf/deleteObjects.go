package receiptPdf

import (
	"aws_h"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"log"
	"time"
)

func publishBuildingChange(ctx context.Context, buildingId string) error {
	return nil
}

func DeleteByBuilding(ctx context.Context, buildingId string) error {

	prefix := buildingId + "/"
	return DeleteObjects(ctx, &prefix)
}

func DeleteByReceipt(ctx context.Context, buildingId string, receiptId int32, dateTime time.Time) error {

	date := dateTime.Format(time.DateOnly)
	prefix := fmt.Sprintf("%s/%s/%d/", buildingId, date, receiptId)
	return DeleteObjects(ctx, &prefix)
}

func DeleteObjects(ctx context.Context, prefix *string) error {
	bucket, err := resource.Get("ReceiptsBucket", "name")
	if err != nil {
		return err
	}

	bucketName := bucket.(string)

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return err
	}

	s3List, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: prefix,
	})

	if err != nil {
		return err
	}

	if len(s3List.Contents) == 0 {
		return nil
	}

	log.Printf("Objects: %d", len(s3List.Contents))

	return nil
}
