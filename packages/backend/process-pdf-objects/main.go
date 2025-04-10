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

	log.Printf("Processing %d records", len(sqsEvent.Records))

	for _, sqsRecord := range sqsEvent.Records {
		log.Printf("EventSource %s", sqsRecord.EventSource)
		var event receiptPdf.QueueEvent

		err := json.Unmarshal([]byte(sqsRecord.Body), &event)
		if err != nil {
			return err
		}

		if event.IsChanges() {
			err = receiptPdf.DeleteByEvent(ctx, event)
			if err != nil {
				return err
			}

			continue
		}

		if event.Type == receiptPdf.SendPdfs {
			if event.ProgressId == "" {
				log.Printf("ProgressId is empty")
				continue
			}

			holder := Holder{
				ctx:     ctx,
				event:   event,
				Subject: event.Subject,
				Message: event.Message,
			}

			err = holder.sendPdfs()
			if err != nil {
				return err
			}

			continue
		}

	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
