package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/yaz/kyo-repo/internal/util"
)

func (service Service) StartUrl(userId string) (string, error) {
	holder, err := GetTelegramBot()
	if err != nil {
		return "", err
	}

	user, err := holder.B.GetMe(service.ctx)
	if err != nil {
		return "", fmt.Errorf("GetWebhook me error: %w", err)
	}

	log.Printf("Telegram bot user: %v", user)

	return fmt.Sprintf("https://t.me/%s?start=%s", user.Username, userId), nil
}

func (service Service) Info() (*Info, error) {

	holder, err := GetTelegramBot()
	if err != nil {
		return nil, err
	}

	info := &Info{}

	userBot, err := holder.B.GetMe(service.ctx)
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
	holder, err := GetTelegramBot()
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
			_, err := holder.B.SendMessage(service.ctx, &bot.SendMessageParams{
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

func (h Holder) GetProfilePictures(ctx context.Context, chatId int64) ([]ProfilePicture, error) {
	array := make([]ProfilePicture, 0)

	profilePhotos, err := h.B.GetUserProfilePhotos(ctx, &bot.GetUserProfilePhotosParams{UserID: chatId})
	if err != nil {
		return array, fmt.Errorf("GetUserProfilePhotos error: %w", err)
	}

	if profilePhotos.TotalCount > 0 {

		length := 0
		for _, photoSizes := range profilePhotos.Photos {
			length += len(photoSizes)
		}

		var wg sync.WaitGroup
		wg.Add(length)

		pictureChan := make(chan ProfilePicture, length)
		errorChan := make(chan error, length)

		for _, photoSizes := range profilePhotos.Photos {
			for _, photo := range photoSizes {
				go func() {
					defer wg.Done()
					file, err := h.B.GetFile(ctx, &bot.GetFileParams{FileID: photo.FileID})
					if err != nil {
						log.Printf("Error getting telegram profile photo file: %v", err)
						errorChan <- err
						return

					}
					link := h.B.FileDownloadLink(file)
					log.Printf("Profile photo link: %s", link)
					pictureChan <- ProfilePicture{
						FileID:       file.FileID,
						FileUniqueID: file.FileUniqueID,
						FileSize:     file.FileSize,
						FilePath:     file.FilePath,
						FileLink:     link,
						Width:        photo.Width,
						Height:       photo.Height,
					}
				}()
			}
		}

		wg.Wait()
		close(errorChan)
		close(pictureChan)

		err = util.HasErrors(errorChan)
		if err != nil {
			return nil, fmt.Errorf("GetProfilePictures error: %w", err)
		}

		for picture := range pictureChan {
			array = append(array, picture)
		}
	}

	return array, nil
}
