package telegram

import (
	"fmt"
	"github.com/sst/sst/v3/sdk/golang/resource"
)

func GetWebhookTelegramBotApiKey() (string, error) {
	apiKey, err := resource.Get("TelegramBotApiKey", "value")
	if err != nil {
		return "", fmt.Errorf("GetWebhookTelegramBotApiKey error: %w", err)
	}

	return apiKey.(string), nil
}
