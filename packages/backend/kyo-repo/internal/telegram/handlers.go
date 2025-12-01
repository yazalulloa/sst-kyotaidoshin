package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/backup"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/receipts"
	"github.com/yaz/kyo-repo/internal/users"
	"github.com/yaz/kyo-repo/internal/util"
)

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	userId := strings.TrimSpace(strings.ReplaceAll(update.Message.Text, _START_COMMAND, ""))
	if userId == "" {
		log.Printf("userId is empty")
		return
	}

	chat := update.Message.Chat
	rows, err := users.NewRepository(ctx).UpdateTelegramChat(userId, chat.ID, chat.Username, chat.FirstName, chat.LastName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("user %s not found", userId)
			return
		}

		log.Printf("Error getting user: %s", err)
		return
	}

	log.Printf("Rows affected: %d", rows)

	//from := update.Message.From

	msgParams := &bot.SendMessageParams{
		ChatID: chat.ID,
		Text:   "Cuenta enlazada",
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	}

	_, err = b.SendMessage(ctx, msgParams)

	if err != nil {
		log.Printf("Error sending message: %v\n%s", msgParams, err)
		return
	}
}

func optionsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	chat := update.Message.Chat

	var dest []model.Permissions

	stmt := Permissions.SELECT(Permissions.AllColumns).
		FROM(
			TelegramChats.
				INNER_JOIN(Users, TelegramChats.UserID.EQ(Users.ID)).
				INNER_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
				INNER_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)).
				INNER_JOIN(RolePermissions, Roles.ID.EQ(RolePermissions.RoleID)).
				INNER_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
		).
		WHERE(TelegramChats.ChatID.EQ(sqlite.Int64(chat.ID)))

	err := stmt.QueryContext(ctx, db.GetDB().DB, &dest)

	if err != nil {
		log.Printf("Error getting permissions: %v", err)
		return
	}

	log.Printf("Perms: %d", len(dest))

	var msgParams *bot.SendMessageParams

	if len(dest) == 0 {
		msgParams = &bot.SendMessageParams{
			ChatID: chat.ID,
			Text:   "No tienes opciones registradas",
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		}

	} else {
		keyboardMarkup := models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         "Ultima tasa de cambio",
						CallbackData: _LAST_RATE_CALLBACK,
					},
				},
				{
					{
						Text:         "Recibos",
						CallbackData: _RECEIPTS_CALLBACK,
					},
				},
				{
					{
						Text:         "Backups",
						CallbackData: _BACKUPS_CALLBACK,
					},
				},
			},
		}

		msgParams = &bot.SendMessageParams{
			ChatID:      chat.ID,
			Text:        "Elige una opción",
			ReplyMarkup: keyboardMarkup,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		}
	}

	_, err = b.SendMessage(ctx, msgParams)

	if err != nil {
		log.Printf("Error sending message: %v\n%s", msgParams, err)
		return
	}
	return

}

func tasaHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	array, err := rates.NewRepository(ctx).LastRate(util.USD.Name(), "EUR")
	if err != nil {
		log.Printf("Error getting last rate: %v", err)
		return
	}

	chat := update.Message.Chat

	var builder strings.Builder

	for i, rate := range array {
		builder.WriteString(rateMsg(rate))
		if i < len(array)-1 {
			builder.WriteString("\n")
		}
	}

	msgParams := &bot.SendMessageParams{
		ChatID: chat.ID,
		Text:   builder.String(),
		//ShowAlert:       false, //show modal
		//CacheTime:       0,
	}

	_, err = b.SendMessage(ctx, msgParams)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func flagEmoji(str string) string {
	switch str {
	case "USD":
		return "🇺🇸"
	case "EUR":
		return "🇪🇺"
	default:
		return ""
	}
}

func rateMsg(rate model.Rates) string {
	return fmt.Sprintf("BCV %s  %s  %s", flagEmoji(rate.FromCurrency), util.VED.Format(rate.Rate), rate.DateOfRate.Format(time.DateOnly))
}

func SendRate(ctx context.Context, rate model.Rates) {

	msg := rateMsg(rate)

	list, err := users.NewRepository(ctx).GetTelegramIdsByNotificationEvent(users.NEW_RATE)
	if err != nil {
		log.Printf("Error getting telegram ids for new rate notification: %v", err)
		return
	}
	length := len(list)
	if length == 0 {
		return
	}

	b, err := GetTelegramBot()
	if err != nil {
		log.Printf("Error getting telegram bot: %v", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(length)
	errorChan := make(chan error, length)

	for _, chatId := range list {
		go func() {
			defer wg.Done()
			msgParams := &bot.SendMessageParams{
				ChatID: chatId,
				Text:   msg,
			}

			_, err = b.SendMessage(ctx, msgParams)

			if err != nil {
				errorChan <- fmt.Errorf("error sending message: %v", err)
				return
			}
		}()
	}

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
}

func lastRateCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	array, err := rates.NewRepository(ctx).LastRate(util.USD.Name(), "EUR")
	if err != nil {
		log.Printf("Error getting last rate: %v", err)
		return
	}

	var builder strings.Builder

	for i, rate := range array {
		builder.WriteString(rateMsg(rate))
		if i < len(array)-1 {
			builder.WriteString("\n")
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		msgParams := &bot.SendMessageParams{
			ChatID: update.CallbackQuery.From.ID,
			Text:   builder.String(),
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		}

		_, err = b.SendMessage(ctx, msgParams)

		if err != nil {
			errorChan <- fmt.Errorf("error sending message: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
}

func receiptsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	ids, err := buildings.NewRepository(ctx).SelectIds()
	if err != nil {
		log.Printf("Error getting buildings ids: %v", err)
		return
	}

	options := make([][]models.InlineKeyboardButton, len(ids)+1)

	for i, id := range ids {

		options[i] = []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("Edificio %s", id),
				CallbackData: _RECEIPTS_BUILDING_CALLBACK + id,
			},
		}
	}

	options[len(ids)] = []models.InlineKeyboardButton{
		{
			Text:         "Ultimos recibos",
			CallbackData: "receipts_last",
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.From.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      "Choose an option",
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: options,
			},
		})

		if err != nil {
			errorChan <- fmt.Errorf("error editing message text: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

}

func receiptsBuildingCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	buildingId := strings.TrimPrefix(update.CallbackQuery.Data, _RECEIPTS_BUILDING_CALLBACK)
	if buildingId == "" {
		log.Printf("Building ID is empty in callback data: %s", update.CallbackQuery.Data)
		return
	}

	res, err := receipts.NewService(ctx).GetTableResponse(receipts.RequestQuery{
		Buildings: []string{buildingId},
		SortOrder: util.SortOrderTypeDESC,
		Limit:     11,
	})

	if err != nil {
		log.Printf("Error getting receipts list: %v", err)
		return
	}

	list := res.Results

	options := make([][]models.InlineKeyboardButton, len(list)+1)

	if len(list) == 0 {
		options[0] = []models.InlineKeyboardButton{
			{
				Text:         "No hay recibos",
				CallbackData: _RECEIPTS_CALLBACK,
			},
		}

	} else {
		for i := 0; i < len(list)-1; i++ {
			receipt := list[i].Item

			options[i] = []models.InlineKeyboardButton{
				{
					Text:         fmt.Sprintf("%s %d %s", util.FromInt16ToMonth(receipt.Month), receipt.Year, receipt.Date.Format(time.DateOnly)),
					CallbackData: fmt.Sprintf("%s%s_%s", _RECEIPT_LIST_APT_CALLBACK, buildingId, receipt.ID),
				},
				{
					Text:         "ZIP",
					CallbackData: fmt.Sprintf("%s%s_%s", _RECEIPT_ZIP_CALLBACK, buildingId, receipt.ID),
				},
			}
		}

		options[len(list)-1] = []models.InlineKeyboardButton{
			{
				Text:         "Back",
				CallbackData: _RECEIPTS_CALLBACK,
			},
		}

		options[len(list)] = []models.InlineKeyboardButton{
			{
				Text:         "Next",
				CallbackData: _RECEIPTS_BUILDING_CALLBACK + buildingId,
			},
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		params := &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.From.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      fmt.Sprintf("Receipts %s %d", buildingId, *res.Counters.QueryCount),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: options,
			},
		}

		_, err := b.EditMessageText(ctx, params)

		if err != nil {
			errorChan <- fmt.Errorf("error editing message text: %v", err)

			byteArray, err := json.Marshal(params)
			if err != nil {
				log.Printf("error marshaling params: %v", err)
			} else {
				log.Printf("Error edit Params: %s", byteArray)
			}

			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
}

func receiptZipCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	data := strings.TrimPrefix(update.CallbackQuery.Data, _RECEIPT_ZIP_CALLBACK)
	parts := strings.Split(data, "_")
	if len(parts) != 2 {
		log.Printf("Invalid callback data: %s", update.CallbackQuery.Data)
		return
	}

	buildingId := parts[0]
	receiptId := parts[1]

	zipInfo, err := receipts.GetZipObjectKey(ctx, buildingId, receiptId)
	if err != nil {
		log.Printf("Error getting receipt zip: %v", err)
		return
	}

	fileName := zipInfo.ObjectKey[strings.LastIndex(zipInfo.ObjectKey, "/")+1:]
	filePath := zipInfo.FilePath
	if filePath == "" {

		filePath = util.TmpFileName(util.UuidV7() + fileName)
		err := aws_h.WriteObjectToDisk(ctx, zipInfo.BucketName, zipInfo.ObjectKey, filePath)
		if err != nil {
			log.Printf("Error getting receipt zip from S3: %v", err)
			return
		}
	}

	answerCallbackWithDocument(filePath, fileName, ctx, b, update)
}

func receiptListAptCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {
	data := strings.TrimPrefix(update.CallbackQuery.Data, _RECEIPT_LIST_APT_CALLBACK)
	if data == "" {
		log.Printf("Invalid callback data: %s", update.CallbackQuery.Data)
		return
	}

	parts := strings.Split(data, "_")
	if len(parts) != 2 {
		log.Printf("Invalid callback data: %s", update.CallbackQuery.Data)
		return
	}

	buildingId := parts[0]
	receiptId := parts[1]

	receipt, err := receipts.NewRepository(ctx).SelectById(receiptId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Receipt %s not found", receiptId)
			return
		}
		log.Printf("Error getting receipt: %v", err)
		return
	}

	apts, err := apartments.NewRepository(ctx).SelectByBuilding(buildingId)
	if err != nil {
		log.Printf("Error getting receipt apartments: %v", err)
		return
	}

	options := make([][]models.InlineKeyboardButton, len(apts)+2)

	options[0] = []models.InlineKeyboardButton{
		{
			Text:         "Back",
			CallbackData: _RECEIPTS_BUILDING_CALLBACK + buildingId,
		},
	}

	for i, apt := range apts {
		options[i+1] = []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s\t%s", apt.Number, apt.Name),
				CallbackData: fmt.Sprintf("%s%s_%s_%s", _RECEIPT_PDF_APT, buildingId, receiptId, apt.Number),
			},
		}
	}

	options[len(apts)+1] = []models.InlineKeyboardButton{
		{
			Text:         "ZIP",
			CallbackData: fmt.Sprintf("%s%s_%s", _RECEIPT_ZIP_CALLBACK, buildingId, receiptId),
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		text := fmt.Sprintf("Receipt %s %s %d %s\nApts: %d", buildingId, util.FromInt16ToMonth(receipt.Month),
			receipt.Year, receipt.Date.Format(time.DateOnly), len(apts))

		params := &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.From.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      text,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: options,
			},
		}

		_, err := b.EditMessageText(ctx, params)

		if err != nil {
			errorChan <- fmt.Errorf("error editing message text: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
}

func receiptPdfAptCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	data := strings.TrimPrefix(update.CallbackQuery.Data, _RECEIPT_PDF_APT)
	if data == "" {
		log.Printf("Invalid callback data: %s", update.CallbackQuery.Data)
		return
	}

	parts := strings.Split(data, "_")
	if len(parts) != 3 {
		log.Printf("Invalid callback data: %s", update.CallbackQuery.Data)
		return
	}

	buildingId := parts[0]
	receiptId := parts[1]
	aptNumber := parts[2]

	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		log.Printf("Error getting receipts bucket: %v", err)
		return
	}

	receipt, err := receipts.CalculateReceipt(ctx, buildingId, receiptId)
	if err != nil {
		log.Printf("Error calculating receipt: %v", err)
		return
	}

	keys := receipts.DownloadKeys{
		BuildingId: buildingId,
		Id:         receiptId,
		Parts:      []string{aptNumber},
	}

	partsInfo, err := receipts.GetParts(receipt, ctx, true, &keys)
	if err != nil {
		log.Printf("Error getting receipt parts: %v", err)
		return
	}

	objectKey := partsInfo[0].ObjectKey
	fileName := fmt.Sprintf("%s_%d_%s_%s_%s.pdf", buildingId, receipt.Receipt.Year,
		util.FromInt16ToMonth(receipt.Receipt.Month), aptNumber, receipt.Receipt.Date.Format(time.DateOnly))
	filepath := util.TmpFileName(fmt.Sprintf("receipt_%s_%s", util.UuidV7(), fileName))
	err = aws_h.WriteObjectToDisk(ctx, bucketName, objectKey, filepath)
	if err != nil {
		log.Printf("Error getting receipt pdf from S3: %v", err)
		return
	}

	answerCallbackWithDocument(filepath, fileName, ctx, b, update)

}

func backupsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.From.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      "Choose a backup",
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         "Apartments",
							CallbackData: _BACKUP_APARTMENTS_CALLBACK,
						},
					},
					{
						{
							Text:         "Buildings",
							CallbackData: _BACKUP_BUILDINGS_CALLBACK,
						},
					},
					{
						{
							Text:         "Receipts",
							CallbackData: _BACKUP_RECEIPTS_CALLBACK,
						},
					},
					{
						{
							Text:         "All",
							CallbackData: _BACKUP_ALL_CALLBACK,
						},
					},
				},
			},
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

}

func answerCallbackWithDocument(filepath, filename string, ctx context.Context, b *bot.Bot, update *models.Update) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Printf("Error reading apartments backup file: %v", err)
		return
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file: %s", err)
			return
		}

		err = os.Remove(filepath)
		if err != nil {
			log.Printf("Error removing file: %s", err)
			return
		}

	}(file)

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID: update.CallbackQuery.From.ID,
			Document: &models.InputFileUpload{
				Filename: filename,
				Data:     file,
			},
		})

		if err != nil {
			errorChan <- fmt.Errorf("error sending apartments backup: %v", err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			//Text:            msg,
			//ShowAlert:       false, //show modal
			//CacheTime:       0,
		})

		if err != nil {
			errorChan <- fmt.Errorf("error answering callback query: %v", err)
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
}

func backupApartmentsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	filepath, err := apartments.NewService(ctx).Backup()
	if err != nil {
		log.Printf("Error getting apartments backup: %v", err)
		return
	}
	defer func() {
		err := os.Remove(filepath)
		if err != nil {
			log.Printf("Error removing file: %s", err)
			return
		}
	}()

	answerCallbackWithDocument(filepath, api.BACKUP_APARTMENTS_FILE, ctx, b, update)
}

func backupBuildingsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	filepath, err := buildings.NewService(ctx).Backup()
	if err != nil {
		log.Printf("Error getting buildings backup: %v", err)
		return
	}
	defer func() {
		err := os.Remove(filepath)
		if err != nil {
			log.Printf("Error removing file: %s", err)
			return
		}
	}()

	answerCallbackWithDocument(filepath, api.BACKUP_BUILDINGS_FILE, ctx, b, update)
}

func backupReceiptsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {
	filepath, err := receipts.NewService(ctx).Backup()
	if err != nil {
		log.Printf("Error getting receipts backup: %v", err)
		return
	}
	defer func() {
		err := os.Remove(filepath)
		if err != nil {
			log.Printf("Error removing file: %s", err)
			return
		}
	}()

	answerCallbackWithDocument(filepath, api.BACKUP_RECEIPTS_FILE, ctx, b, update)
}

func backupAllCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	filepath, err := backup.AllBackup(ctx)
	if err != nil {
		log.Printf("Error getting all backup: %v", err)
		return
	}
	defer func() {
		err := os.Remove(filepath)
		if err != nil {
			log.Printf("Error removing file: %s", err)
			return
		}
	}()

	answerCallbackWithDocument(filepath, api.BACKUP_ALL_FILE, ctx, b, update)
}
