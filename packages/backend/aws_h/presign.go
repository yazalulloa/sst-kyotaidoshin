package aws_h

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"time"
)

func PresignGet(ctx context.Context, bucketName, objectKey string, expires time.Duration) (string, error) {

	client, err := GetPresignClient(ctx)
	if err != nil {
		return "", err
	}

	req, err := client.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(options *s3.PresignOptions) {
		options.Expires = expires
	})

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
