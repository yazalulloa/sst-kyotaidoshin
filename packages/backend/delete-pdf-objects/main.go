package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"kyotaidoshin/receiptPdf"
	"kyotaidoshin/receipts"
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

		if event.Type == receiptPdf.BuildPdfs {
			receipt, err := receipts.CalculateReceipt(event.BuildingId, event.ReceiptId)
			if err != nil {
				return err
			}

			parts, err := receipts.GetParts(receipt, ctx, &receipts.DownloadKeys{IsApt: true})
			if err != nil {
				return err
			}

			log.Printf("Parts %d", len(parts))

			for _, part := range parts {
				req := receiptPdf.SendPdfRequest{
					//Emails:        strings.Split(part.Apt.Emails, ","),
					Emails:        []string{"yzlup2@gmail.com"},
					MonthStr:      receipt.MonthStr,
					Year:          receipt.Receipt.Year,
					BuildingName:  receipt.Building.Name,
					AptNumber:     part.Apt.Number,
					SubjectPrefix: "",
					Text:          "",
					ObjectKey:     part.ObjectKey,
					EmailKey:      receipt.Building.EmailConfig,
				}

				_, err = receiptPdf.SendPdf(ctx, req)
				if err != nil {
					return err
				}
			}

			log.Printf("Send %d", len(parts))
			_, err = receipts.UpdateLastSent(event.ReceiptId)
			if err != nil {
				return err
			}
			continue
		}

	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
