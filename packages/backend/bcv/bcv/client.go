package bcv

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"os"
	"sync"
)

func loadConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx, func(opts *config.LoadOptions) error {
		opts.Region = os.Getenv("AWS_REGION")
		return nil
	})
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
