package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/receipts"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"strings"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {

	for _, sqsRecord := range sqsEvent.Records {
		//fmt.Printf("The sqsRecord %s for event source %s = %s \n", sqsRecord.MessageId, sqsRecord.EventSource, sqsRecord.Body)
		var s3Event events.S3Event
		err := json.Unmarshal([]byte(sqsRecord.Body), &s3Event)
		if err != nil {
			return err
		}

		for _, s3Record := range s3Event.Records {
			log.Printf("S3 Event %s", s3Record.EventName)

			if !strings.Contains(s3Record.EventName, "ObjectCreated:Post") {
				log.Printf("Skipping %s", s3Record.S3.Object.Key)
				continue
			}

			backupType, err := util.GetBackupTypeStartsWith(strings.ToUpper(s3Record.S3.Object.Key))
			if err != nil {
				log.Printf("Error getting backup type: %s", err)
				return err
			}

			var processDecoder func(*json.Decoder) (int64, error)

			switch backupType {
			case util.APARTMENTS:
				processDecoder = apartments.NewService(ctx).ProcessDecoder
				break
			case util.BUILDINGS:
				processDecoder = buildings.ProcessDecoder
				break
			case util.RECEIPTS:
				processDecoder = receipts.ProcessDecoder
				break
			}

			inserted, err := api.ProcessBackup(ctx, &s3Record.S3.Bucket.Name, &s3Record.S3.Object.Key, nil, processDecoder)
			if err != nil {
				return err
			}

			log.Printf("S3 Event Processed %s records %d", s3Record.S3.Object.Key, inserted)
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
