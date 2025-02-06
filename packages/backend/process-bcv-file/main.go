package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"process-bcv-file/file"
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
			fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", s3Record.EventSource, s3Record.EventTime, s3Record.S3.Bucket.Name, s3Record.S3.Object.Key)
			err := file.ParseFile(ctx, s3Record.S3.Bucket.Name, s3Record.S3.Object.Key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
