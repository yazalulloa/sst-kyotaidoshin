package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"kyo-repo/internal/apartments"
	"kyo-repo/internal/api"
	"kyo-repo/internal/backup"
	"kyo-repo/internal/buildings"
	"kyo-repo/internal/db"
	"kyo-repo/internal/db/gen/model"
	. "kyo-repo/internal/db/gen/table"
	"kyo-repo/internal/rates"
	"kyo-repo/internal/receipts"
	"kyo-repo/internal/users"
	"kyo-repo/internal/util"
	"log"
	"os"
	"strings"
	"sync"
	"time"
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
						CallbackData: "receipts",
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

func lastRateCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {
	//byteArray, err := json.MarshalIndent(update, "", "  ")
	//if err != nil {
	//	log.Printf("Error marshalling update: %s", err)
	//	return
	//}
	//
	//log.Printf("Update callback: %s", byteArray)

	location, err := util.TzCss()
	if err != nil {
		log.Printf("Error getting timezone: %v", err)
		return
	}

	rate, err := rates.LastRate(util.USD.Name())
	if err != nil {
		log.Printf("Error getting last rate: %v", err)
		return
	}

	msg := fmt.Sprintf("TASA: %s\nFECHA: %s\nCREADO: %s", util.VED.Format(rate.Rate),
		rate.DateOfRate.Format(time.DateOnly), rate.CreatedAt.In(location).Format(time.DateTime))

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		msgParams := &bot.SendMessageParams{
			ChatID: update.CallbackQuery.From.ID,
			Text:   msg,
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

func sendBackup(filepath, filename string, ctx context.Context, b *bot.Bot, update *models.Update) {
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

	filepath, err := apartments.Backup()
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

	sendBackup(filepath, api.BACKUP_APARTMENTS_FILE, ctx, b, update)
}

func backupBuildingsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	filepath, err := buildings.Backup()
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

	sendBackup(filepath, api.BACKUP_BUILDINGS_FILE, ctx, b, update)
}

func backupReceiptsCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {
	filepath, err := receipts.Backup()
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

	sendBackup(filepath, api.BACKUP_RECEIPTS_FILE, ctx, b, update)
}

func backupAllCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {

	filepath, err := backup.AllBackup(api.BACKUP_ALL_FILE)
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

	sendBackup(filepath, api.BACKUP_ALL_FILE, ctx, b, update)
}
