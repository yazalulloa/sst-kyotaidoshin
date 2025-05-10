package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/file"
	"log"
	"strings"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	//eventJson, _ := json.MarshalIndent(sqsEvent, "", "    ")
	//log.Printf("EVENT: %s", eventJson)

	for _, sqsRecord := range sqsEvent.Records {
		//fmt.Printf("The sqsRecord %s for event source %s = %s \n", sqsRecord.MessageId, sqsRecord.EventSource, sqsRecord.Body)
		var s3Event events.S3Event
		err := json.Unmarshal([]byte(sqsRecord.Body), &s3Event)
		if err != nil {
			return err
		}

		for _, s3Record := range s3Event.Records {
			log.Printf("S3 Event %s", s3Record.EventName)

			if strings.Contains(s3Record.EventName, "ObjectCreated:Copy") {
				log.Printf("Skipping %s", s3Record.S3.Object.Key)
				continue
			}

			err := file.ParseFile(file.ParsingParams{
				Ctx:    ctx,
				Bucket: s3Record.S3.Bucket.Name,
				Key:    s3Record.S3.Object.Key,
			})

			if err != nil {
				return err
			}
			log.Printf("S3 Event Processed %s", s3Record.S3.Object.Key)
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
