package telegram

import (
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sst/sst/v3/sdk/golang/resource"
)

func (service Service) SetWebhook() error {

	telegramBot, err := GetTelegramBot()
	if err != nil {
		return err
	}

	apiKey, err := GetWebhookTelegramBotApiKey()
	if err != nil {
		return err
	}

	functionUrl, err := resource.Get("TelegramWebhookFunction", "url")

	if err != nil {
		return fmt.Errorf("GetWebhook TelegramWebhookFunction error: %w", err)
	}

	ok, err := telegramBot.SetWebhook(service.ctx, &bot.SetWebhookParams{
		URL:            functionUrl.(string),
		MaxConnections: 1,
		SecretToken:    apiKey,
	})

	if err != nil {
		return fmt.Errorf("SetWebhook webhook error: %w", err)
	}

	if !ok {
		return fmt.Errorf("SetWebhook webhook failed")
	}

	return nil
}

func (service Service) DeleteWebhook() error {
	telegramBot, err := GetTelegramBot()
	if err != nil {
		return err
	}

	ok, err := telegramBot.DeleteWebhook(service.ctx, &bot.DeleteWebhookParams{
		DropPendingUpdates: true,
	})

	if err != nil {
		return fmt.Errorf("DeleteWebhook webhook error: %w", err)
	}

	if !ok {
		return fmt.Errorf("DeleteWebhook webhook failed")
	}

	return nil
}

func (service Service) GetWebhook() (*models.WebhookInfo, error) {
	telegramBot, err := GetTelegramBot()
	if err != nil {
		return nil, err
	}

	webhook, err := telegramBot.GetWebhookInfo(service.ctx)
	if err != nil {
		return nil, fmt.Errorf("GetWebhook webhook error: %w", err)
	}

	return webhook, nil

}
