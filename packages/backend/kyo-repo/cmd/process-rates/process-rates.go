package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(process)
}

func process(ctx context.Context, event events.SQSEvent) (string, error) {

	for _, record := range event.Records {

		dest := RateRequest{}
		err := json.Unmarshal([]byte(record.Body), &dest)
		if err != nil {
			return "", err
		}

		log.Printf("Processing %d rates", len(dest.rates))

		//if len(dest.rates) == 0 {
		//	log.Printf("No rates to process %s", record.Body)
		//}
	}

	return "OK", nil
}

type RateRequest struct {
	rates []RateItem
}
type RateItem struct {
	Id            int64   `json:"id"`
	FromCurrency  string  `json:"from_currency"`
	Rate          float64 `json:"rate"`
	DateOfRate    string  `json:"date_of_rate"`
	DateOfFile    *string `json:"date_of_file"`
	AltDateOfFile *string `json:"alt_date_of_file"`
}
