package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/bcv"
	"log"
)

func handler(ctx context.Context, event interface{}) (string, error) {
	err := bcv.Check(ctx)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(handler)
}
