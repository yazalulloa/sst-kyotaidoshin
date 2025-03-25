package telegram_webhook

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func handler(ctx context.Context, str string) (string, error) {
	log.Printf("Received: %s", str)
	return "", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}
