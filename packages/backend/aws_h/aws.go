package aws_h

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"log"
	"os"
	"sync"
	"time"
)

var configInstance *aws.Config
var configOnce sync.Once

func loadConfig(ctx context.Context) (aws.Config, error) {
	var err error

	configOnce.Do(func() {
		awsConfig, err := config.LoadDefaultConfig(ctx, func(opts *config.LoadOptions) error {
			opts.Region = os.Getenv("AWS_REGION")
			return nil
		})

		if err == nil {
			configInstance = &awsConfig
		}
	})

	return *configInstance, err
}

var s3ClientInstance *s3.Client
var s3ClientOnce sync.Once

func GetS3Client(ctx context.Context) (*s3.Client, error) {

	var err error
	s3ClientOnce.Do(func() {
		s3ClientInstance, err = s3client(ctx)
	})

	return s3ClientInstance, err
}

func s3client(ctx context.Context) (*s3.Client, error) {

	cfg, err := loadConfig(ctx)

	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)

	return client, nil
}

var lambdaClientInstance *lambda.Client
var lambdaClientOnce sync.Once

func GetLambdaClient(ctx context.Context) (*lambda.Client, error) {
	var err error
	lambdaClientOnce.Do(func() {
		lambdaClientInstance, err = lambdaClient(ctx)
	})

	return lambdaClientInstance, err
}

func lambdaClient(ctx context.Context) (*lambda.Client, error) {

	cfg, err := loadConfig(ctx)

	if err != nil {
		return nil, err
	}
	client := lambda.NewFromConfig(cfg)

	return client, nil
}

var presignClientInstance *s3.PresignClient
var presignClientOnce sync.Once

func GetPresignClient(ctx context.Context) (*s3.PresignClient, error) {
	var err error
	presignClientOnce.Do(func() {
		presignClientInstance, err = presignClient(ctx)
	})

	return presignClientInstance, err
}

func presignClient(ctx context.Context) (*s3.PresignClient, error) {

	cfg, err := loadConfig(ctx)

	if err != nil {
		return nil, err
	}
	client := s3.NewPresignClient(s3.NewFromConfig(cfg))

	return client, nil
}

func PresignPostObject(ctx context.Context, bucketName string, objectKey string, optionFn func(options *s3.PresignPostOptions)) (*s3.PresignedPostRequest, error) {

	client, err := GetPresignClient(ctx)
	if err != nil {
		log.Printf("Couldn't get a presigned post request to put %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		return nil, err
	}

	request, err := client.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, optionFn)
	if err != nil {
		log.Printf("Couldn't get a presigned post request to put %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		return nil, err
	}
	return request, nil
}

func PresignPut(ctx context.Context, bucketName, objectKey, contentType string) (string, error) {
	client, err := GetPresignClient(ctx)
	if err != nil {
		return "", err
	}

	optionFn := func(options *s3.PresignOptions) {
		options.Expires = 2 * time.Hour
	}

	req, err := client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, optionFn)

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

var sqsClientInstance *sqs.Client
var sqsClientOnce sync.Once

func GetSqsClient(ctx context.Context) (*sqs.Client, error) {
	var err error
	sqsClientOnce.Do(func() {
		sqsClientInstance, err = sqsclient(ctx)
	})

	return sqsClientInstance, err
}

func sqsclient(ctx context.Context) (*sqs.Client, error) {

	cfg, err := loadConfig(ctx)

	if err != nil {
		return nil, err
	}
	client := sqs.NewFromConfig(cfg)

	return client, nil
}
