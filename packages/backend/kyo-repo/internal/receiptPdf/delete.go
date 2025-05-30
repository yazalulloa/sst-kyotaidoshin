package receiptPdf

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
)

func DeleteByBuilding(ctx context.Context, buildingId string) error {

	prefix := fmt.Sprintf("RECEIPTS/%s/", buildingId)
	return DeleteObjects(ctx, &prefix)
}

func DeleteByReceipt(ctx context.Context, buildingId string, receiptId string) error {

	prefix := fmt.Sprintf("RECEIPTS/%s/%s/", buildingId, receiptId)
	return DeleteObjects(ctx, &prefix)
}

func DeleteByEvent(ctx context.Context, event QueueEvent) error {

	if event.Type == BuildingChanges {
		return DeleteByBuilding(ctx, event.BuildingId)
	}

	return DeleteByReceipt(ctx, event.BuildingId, event.ReceiptId)
}

func DeleteObjects(ctx context.Context, prefix *string) error {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return err
	}

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

	str := ""

	if prefix != nil {
		str = *prefix
	}

	if len(s3List.Contents) == 0 {

		log.Printf("No objects found in bucket %s with prefix %s", bucketName, str)
		return nil
	}

	log.Printf("Deleting %d objects from bucket %s with prefix %s", len(s3List.Contents), bucketName, str)

	list := make([]types.ObjectIdentifier, len(s3List.Contents))
	for i, item := range s3List.Contents {

		list[i] = types.ObjectIdentifier{
			Key: item.Key,
			//ETag:             item.ETag,
			//LastModifiedTime: item.LastModified,
			//Size:             item.Size,
		}
	}

	delOut, err := s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{
			Objects: list,
			Quiet:   aws.Bool(true),
		},
	})

	if err != nil {
		log.Printf("Error deleting objects from bucket %s: %s", bucketName, err)
		return err
	}

	if len(delOut.Errors) > 0 {
		multiErr := &util.MultiError{Errors: make([]error, len(delOut.Errors))}
		for _, outErr := range delOut.Errors {
			multiErr.Add(fmt.Errorf("%s: %s\n", *outErr.Key, *outErr.Message))
		}

		return err
	}

	return nil
}
