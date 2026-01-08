package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/util"
)

type Service struct {
	ctx context.Context
}

func NewService(ctx context.Context) *Service {
	return &Service{ctx: ctx}
}

var _botInstance *Holder
var _telegramBotOnce sync.Once

const _START_COMMAND = "/start"
const _OPTIONS_COMMAND = "/options"
const _TASA_COMMAND = "/tasa"

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

type Holder struct {
	B *bot.Bot
}

func GetTelegramBot() (*Holder, error) {

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

		_b, err := bot.New(token.(string), opts...)

		_b.RegisterHandler(bot.HandlerTypeMessageText, _START_COMMAND, bot.MatchTypePrefix, startHandler)
		_b.RegisterHandler(bot.HandlerTypeMessageText, _OPTIONS_COMMAND, bot.MatchTypeExact, optionsHandler)
		_b.RegisterHandler(bot.HandlerTypeMessageText, _TASA_COMMAND, bot.MatchTypeExact, tasaHandler)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _LAST_RATE_CALLBACK, bot.MatchTypeExact, lastRateCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUPS_CALLBACK, bot.MatchTypeExact, backupsCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPTS_CALLBACK, bot.MatchTypeExact, receiptsCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPTS_BUILDING_CALLBACK, bot.MatchTypePrefix, receiptsBuildingCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_ZIP_CALLBACK, bot.MatchTypePrefix, receiptZipCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_LIST_APT_CALLBACK, bot.MatchTypePrefix, receiptListAptCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _RECEIPT_PDF_APT, bot.MatchTypePrefix, receiptPdfAptCallBack)

		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_APARTMENTS_CALLBACK, bot.MatchTypeExact, backupApartmentsCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_BUILDINGS_CALLBACK, bot.MatchTypeExact, backupBuildingsCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_RECEIPTS_CALLBACK, bot.MatchTypeExact, backupReceiptsCallBack)
		_b.RegisterHandler(bot.HandlerTypeCallbackQueryData, _BACKUP_ALL_CALLBACK, bot.MatchTypeExact, backupAllCallBack)

		if err != nil {
			_err = err
			return
		}

		_botInstance = &Holder{B: _b}

		//log.Printf("Elapsed time: %d", time.Now().UnixMilli()-timestamp)
	})
	return _botInstance, _err
}

type Info struct {
	User    *models.User
	Webhook *models.WebhookInfo
}

type ProfilePicture struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	FileLink     string `json:"file_link,omitempty"`
	CdnPath      string `json:"cdn_path,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}
