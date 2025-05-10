package receiptPdf

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/util"
	"io"
)

type ProgressUpdate struct {
	ObjectKey string `json:"objectKey"`
	Counter   int    `json:"counter"`
	Size      int    `json:"size"`
	Building  string `json:"building"`
	Month     string `json:"month"`
	Date      string `json:"date"`
	Apt       string `json:"apt"`
	Name      string `json:"name"`
	From      string `json:"from"`
	To        string `json:"to"`
	ErrMsg    string `json:"errMsg"`
	Finished  bool   `json:"finished"`
	CardId    string `json:"cardId"`
	Cancelled bool   `json:"cancelled"`

	Etag *string
}

func PutProgress(ctx context.Context, update *ProgressUpdate) error {
	byteArray, err := json.Marshal(*update)
	if err != nil {
		return err
	}

	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return err
	}

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(update.ObjectKey),
		Body:   bytes.NewReader(byteArray),
	})

	if err != nil {
		return err
	}

	return nil
}

func GetProgress(ctx context.Context, objectKey string) (*ProgressUpdate, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return nil, err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	var progress ProgressUpdate
	err = json.NewDecoder(res.Body).Decode(&progress)
	if err != nil {
		return nil, err
	}

	progress.Etag = res.ETag

	return &progress, nil

}

func DeleteProgress(ctx context.Context, objectKey string) error {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return err
	}

	_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return err
	}

	return nil
}
