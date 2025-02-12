package main

import (
	"bcv/bcv"
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"time"
)

func handler(ctx context.Context, event interface{}) (string, error) {

	timestamp := time.Now().UnixMilli()

	if true {
		err := bcv.Check(ctx)
		if err != nil {
			return "", err
		}
		return "OK", nil
	}

	links, err := bcv.AllFiles()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(links)
	if err != nil {
		return "", err
	}

	log.Printf("bcvTask took %d ms", time.Now().UnixMilli()-timestamp)
	return string(jsonBytes), nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(handler)
}
