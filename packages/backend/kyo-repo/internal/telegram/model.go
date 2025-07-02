package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/util"
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
const _LAST_RATE_CALLBACK = "last_rate_callback"
const _BACKUPS_CALLBACK = "backups_callback"
const _RECEIPTS_CALLBACK = "receipts_callback"
const _RECEIPTS_BUILDING_CALLBACK = "rec_b_"
const _RECEIPT_ZIP_CALLBACK = "rec_zip_"
const _RECEIPT_LIST_APT_CALLBACK = "rec_list_apt_"
const _RECEIPT_PDF_APT = "rec_pdf_apt_"

const _BACKUP_APARTMENTS_CALLBACK = "backup_apartments_callback"
const _BACKUP_BUILDINGS_CALLBACK = "backup_buildings_callback"
const _BACKUP_RECEIPTS_CALLBACK = "backup_receipts_callback"
const _BACKUP_ALL_CALLBACK = "backup_all_callback"

func GetTelegramBot() (*bot.Bot, error) {

	var _err error
	_telegramBotOnce.Do(func() {
		//timestamp := time.Now().UnixMilli()
		token, err := resource.Get("TelegramBotToken", "value")
		if err != nil {
			_err = fmt.Errorf("GetWebhook TelegramBotToken error: %w", err)
			return
		}

		apiKey, err := GetWebhookTelegramBotApiKey()
		if err != nil {
			_err = err
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
			bot.WithWebhookSecretToken(apiKey),
		}

		_telegramBotInstance, err = bot.New(token.(string), opts...)

		_telegramBotInstance.RegisterHandler(bot.HandlerTypeMessageText, _START_COMMAND, bot.MatchTypePrefix, startHandler)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeMessageText, _OPTIONS_COMMAND, bot.MatchTypeExact, optionsHandler)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _LAST_RATE_CALLBACK, bot.MatchTypeExact, lastRateCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUPS_CALLBACK, bot.MatchTypeExact, backupsCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPTS_CALLBACK, bot.MatchTypeExact, receiptsCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPTS_BUILDING_CALLBACK, bot.MatchTypePrefix, receiptsBuildingCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_ZIP_CALLBACK, bot.MatchTypePrefix, receiptZipCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_LIST_APT_CALLBACK, bot.MatchTypePrefix, receiptListAptCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_PDF_APT, bot.MatchTypePrefix, receiptPdfAptCallBack)

		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_APARTMENTS_CALLBACK, bot.MatchTypeExact, backupApartmentsCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_BUILDINGS_CALLBACK, bot.MatchTypeExact, backupBuildingsCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_RECEIPTS_CALLBACK, bot.MatchTypeExact, backupReceiptsCallBack)
		_telegramBotInstance.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_ALL_CALLBACK, bot.MatchTypeExact, backupAllCallBack)

		if err != nil {
			_err = err
			return
		}

		//log.Printf("Elapsed time: %d", time.Now().UnixMilli()-timestamp)
	})
	return _telegramBotInstance, _err

}

type Info struct {
	User    *models.User
	Webhook *models.WebhookInfo
}
