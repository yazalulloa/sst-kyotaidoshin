package bcv

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"os"
	"sync"
)

var s3Instance *S3Helper
var s3Once sync.Once

type S3Helper struct {
	Client *s3.Client
}

func GetS3Client(ctx context.Context) (*s3.Client, error) {
	var err error
	s3Once.Do(func() {
		s3Instance, err = s3client(ctx)
	})

	return s3Instance.Client, err
}

func s3client(ctx context.Context) (*S3Helper, error) {

	cfg, err := config.LoadDefaultConfig(ctx, func(opts *config.LoadOptions) error {
		opts.Region = os.Getenv("AWS_REGION")
		return nil
	})

	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)

	return &S3Helper{
		Client: client,
	}, nil
}

var lambdaInstance *LambdaHelper
var lambdaOnce sync.Once

type LambdaHelper struct {
	Client *lambda.Client
}

func GetLambdaClient(ctx context.Context) (*lambda.Client, error) {
	var err error
	lambdaOnce.Do(func() {
		lambdaInstance, err = lambdaClient(ctx)
	})

	return lambdaInstance.Client, err
}

func lambdaClient(ctx context.Context) (*LambdaHelper, error) {

	cfg, err := config.LoadDefaultConfig(ctx, func(opts *config.LoadOptions) error {
		opts.Region = os.Getenv("AWS_REGION")
		return nil
	})

	if err != nil {
		return nil, err
	}
	client := lambda.NewFromConfig(cfg)

	return &LambdaHelper{
		Client: client,
	}, nil
}
