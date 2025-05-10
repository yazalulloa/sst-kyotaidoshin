package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-telegram/bot/models"
	"github.com/yaz/kyo-repo/internal/telegram"
	"log"
	"strings"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (string, error) {

	//if request.HTTPMethod != "POST" {
	//	log.Printf("Invalid HTTP method [%s]", request.HTTPMethod)
	//	return "", nil
	//}

	token := request.Headers["x-telegram-bot-api-secret-token"]

	if token == "" {
		log.Printf("No token provided")
		return "", nil
	}

	body := request.Body

	if body == "" {
		log.Printf("No body provided")
		return "", nil
	}

	update := models.Update{}

	err := json.NewDecoder(strings.NewReader(body)).Decode(&update)
	if err != nil {
		log.Printf("Error decoding body: %s", err)
		return "", err
	}

	telegramBot, err := telegram.GetTelegramBot()
	if err != nil {
		log.Printf("Error getting telegram bot: %s", err)
		return "", err
	}

	telegramBot.ProcessUpdate(ctx, &update)

	return "", nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}
