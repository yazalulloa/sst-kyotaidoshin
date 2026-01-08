package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-telegram/bot/models"
	"github.com/yaz/kyo-repo/internal/telegram"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (string, error) {

	token := request.Headers["x-telegram-bot-api-secret-token"]

	if token == "" {
		log.Printf("No token provided")
		return "", nil
	}

	apiKey, err := telegram.GetWebhookTelegramBotApiKey()
	if err != nil {
		log.Printf("Error getting webhook API key: %s", err)
		return "", err
	}

	if token != apiKey {
		log.Printf("Invalid token provided")
		return "", nil
	}

	body := request.Body

	if body == "" {
		log.Printf("No body provided")
		return "", nil
	}

	update := models.Update{}

	err = json.NewDecoder(strings.NewReader(body)).Decode(&update)
	if err != nil {
		log.Printf("Error decoding body: %s", err)
		return "", err
	}

	holder, err := telegram.GetTelegramBot()
	if err != nil {
		log.Printf("Error getting telegram bot: %s", err)
		return "", err
	}

	holder.B.ProcessUpdate(ctx, &update)

	return "", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}
