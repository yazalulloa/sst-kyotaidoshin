package telegram

import (
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"sync"
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

func (service Service) SendBulkMessage(msg string, chatIds []int64) error {
	telegramBot, err := GetTelegramBot()
	if err != nil {
		return err
	}

	util.RemoveDuplicates(&chatIds)

	var wg sync.WaitGroup
	workers := len(chatIds)
	wg.Add(workers)
	errorChan := make(chan error, workers)

	for _, chatId := range chatIds {
		go func() {
			defer wg.Done()
			_, err := telegramBot.SendMessage(service.ctx, &bot.SendMessageParams{
				ChatID: chatId,
				Text:   msg,
			})
			if err != nil {
				errorChan <- fmt.Errorf("failed to send message to chat %d: %w", chatId, err)
				return
			}
		}()
	}

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return fmt.Errorf("failed to send bulk message: %w", err)
	}

	return nil
}
