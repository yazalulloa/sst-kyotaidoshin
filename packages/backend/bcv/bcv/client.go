package bcv

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"os"
	"sync"
)

var instance *S3Helper
var once sync.Once

type S3Helper struct {
	Client *s3.Client
}

func GetS3Client() (*s3.Client, error) {
	var err error
	once.Do(func() {
		instance, err = client()
	})

	return instance.Client, err
}

func client() (*S3Helper, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), func(opts *config.LoadOptions) error {
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
