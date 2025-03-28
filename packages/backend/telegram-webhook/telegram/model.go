package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"kyotaidoshin/util"
	"log"
	"sync"
	"time"
)

type Service struct {
	ctx context.Context
}

func NewService(ctx context.Context) *Service {
	return &Service{ctx: ctx}
}

var _telegramBotInstance *bot.Bot
var _telegramBotOnce sync.Once

const _START_COMMAND = "/start"
const _OPTIONS_COMMAND = "/options"
const _LAST_RATE_CALLBACK = "last_rate"

func GetTelegramBot() (*bot.Bot, error) {

	var _err error
	_telegramBotOnce.Do(func() {
		timestamp := time.Now().UnixMilli()
		token, err := resource.Get("TelegramBotToken", "value")
		if err != nil {
			_err = fmt.Errorf("GetWebhook TelegramBotToken error: %w", err)
			return
		}

		apiKey, err := resource.Get("TelegramBotApiKey", "value")
		if err != nil {
			_err = fmt.Errorf("GetWebhook TelegramBotApiKey error: %w", err)
			return
		}

		httpClient := util.GetHttpClient()

		opts := []bot.Option{
			bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {
				byteArray, err := json.MarshalIndent(update, "", "  ")
				if err != nil {
					log.Printf("Error marshalling update: %s", err)
					return
				}
				log.Printf("Default handler:\n%s", byteArray)
			}),
			bot.WithHTTPClient(time.Second*10, httpClient),
			bot.WithSkipGetMe(),
			bot.WithNotAsyncHandlers(),
			bot.WithWebhookSecretToken(apiKey.(string)),
		}

		_telegramBotInstance, err = bot.New(token.(string), opts...)

		_telegramBotInstance.RegisterHandler(bot.HandlerTypeMessageText, _START_COMMAND, bot.MatchTypePrefix, startHandler)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeMessageText, _OPTIONS_COMMAND, bot.MatchTypeExact, optionsHandler)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _LAST_RATE_CALLBACK, bot.MatchTypeExact, lastRateCallBack)

		if err != nil {
			_err = err
			return
		}

		log.Printf("Elapsed time: %d", time.Now().UnixMilli()-timestamp)
	})
	return _telegramBotInstance, _err

}

type Info struct {
	User    *models.User
	Webhook *models.WebhookInfo
}
