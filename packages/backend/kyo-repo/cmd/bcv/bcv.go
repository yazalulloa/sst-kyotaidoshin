package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/bcv"
)

func handler(ctx context.Context, event interface{}) (string, error) {
	service, err := bcv.NewService(ctx)
	if err != nil {
		return "", err
	}
	err = service.Check()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(handler)
}
