package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/isr"
)

func handler(ctx context.Context, event interface{}) (string, error) {
	err := isr.UpdateAll(ctx)
	if err != nil {
		log.Printf("Failed to update isr: %v", err)
		return "KO", err
	}
	return "OK", err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}
