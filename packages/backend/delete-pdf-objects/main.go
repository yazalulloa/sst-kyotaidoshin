package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"kyotaidoshin/receiptPdf"
	"log"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {

	for _, sqsRecord := range sqsEvent.Records {
		var event receiptPdf.PublishEvent
		
		err := json.Unmarshal([]byte(sqsRecord.Body), &event)
		if err != nil {
			return err
		}

		err = receiptPdf.DeleteByEvent(ctx, event)
		if err != nil {
			return err
		}

	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
