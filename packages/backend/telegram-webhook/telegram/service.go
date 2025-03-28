package telegram

import (
	"fmt"
	"log"
)

func (service Service) StartUrl(userId string) (string, error) {
	telegramBot, err := GetTelegramBot()
	if err != nil {
		return "", err
	}

	user, err := telegramBot.GetMe(service.ctx)
	if err != nil {
		return "", fmt.Errorf("GetWebhook me error: %w", err)
	}

	log.Printf("Telegram bot user: %v", user)

	return fmt.Sprintf("https://t.me/%s?start=%s", user.Username, userId), nil
}

func (service Service) Info() (*Info, error) {

	telegramBot, err := GetTelegramBot()
	if err != nil {
		return nil, err
	}

	info := &Info{}

	userBot, err := telegramBot.GetMe(service.ctx)
	if err != nil {
		return nil, fmt.Errorf("GetWebhook me error: %w", err)
	}

	info.User = userBot

	webhookInfo, err := service.GetWebhook()
	if err != nil {
		return nil, fmt.Errorf("GetWebhook error: %w", err)
	}

	info.Webhook = webhookInfo

	return info, nil
}
