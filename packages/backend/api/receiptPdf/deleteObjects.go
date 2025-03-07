package receiptPdf

import (
	"aws_h"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"kyotaidoshin/util"
	"log"
)

type Type string

const (
	BUILDING Type = "BUILDING"
	RECEIPT  Type = "RECEIPT"
)

type PublishEvent struct {
	Type       Type   `json:"type"`
	BuildingId string `json:"buildingId"`
	ReceiptId  string `json:"receiptId"`
}

func PublishBuilding(ctx context.Context, buildingId string) {
	event := PublishEvent{
		Type:       RECEIPT,
		BuildingId: buildingId,
		ReceiptId:  "",
	}

	err := publishEvent(ctx, event)
	if err != nil {
		log.Printf("Error publishing building: %v", err)
	}
}

func PublishReceipt(ctx context.Context, buildingId string, receiptId string) {
	event := PublishEvent{
		Type:       BUILDING,
		BuildingId: buildingId,
		ReceiptId:  receiptId,
	}

	err := publishEvent(ctx, event)
	if err != nil {
		log.Printf("Error publishing receipt: %v", err)
	}
}

const messageGroupId = "receiptPdfEvent"

func publishEvent(ctx context.Context, event PublishEvent) error {

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	queue, err := resource.Get("ReceiptPdfChangesQueue", "url")
	if err != nil {
		return err
	}

	client, err := aws_h.GetSqsClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               aws.String(queue.(string)),
		MessageBody:            aws.String(string(bytes)),
		MessageGroupId:         aws.String(messageGroupId),
		MessageDeduplicationId: aws.String(uuid.NewString()),
	})

	if err != nil {
		return err
	}

	return nil
}

func DeleteByBuilding(ctx context.Context, buildingId string) error {

	prefix := buildingId + "/"
	return DeleteObjects(ctx, &prefix)
}

func DeleteByReceipt(ctx context.Context, buildingId string, receiptId string) error {

	prefix := fmt.Sprintf("%s/%s/", buildingId, receiptId)
	return DeleteObjects(ctx, &prefix)
}

func DeleteByEvent(ctx context.Context, event PublishEvent) error {

	if event.Type == BUILDING {
		return DeleteByBuilding(ctx, event.BuildingId)
	}

	return DeleteByReceipt(ctx, event.BuildingId, event.ReceiptId)
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
