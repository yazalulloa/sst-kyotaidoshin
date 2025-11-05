package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	bcv_files "github.com/yaz/kyo-repo/internal/bcv-files"
)

func handler(ctx context.Context, event interface{}) (string, error) {

	err := bcv_files.NewService(ctx).BcvJob()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(handler)
}
