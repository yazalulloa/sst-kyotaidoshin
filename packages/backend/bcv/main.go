package main

import (
	"bcv/bcv"
	"context"
	"github.com/aws/aws-lambda-go/lambda"
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
