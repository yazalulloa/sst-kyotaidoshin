package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yaz/kyo-repo/internal/login"
	"github.com/yaz/kyo-repo/internal/telegram"
	"github.com/yaz/kyo-repo/internal/users"
	"log"
	"time"
)

func handler(ctx context.Context, input login.Input) (*login.UserSubject, error) {

	userInfo, err := func() (*login.UserInfo, error) {
		switch input.Provider {
		case "github":
			return login.GetGithubUserInfo(ctx, input)
		case "google":
			return login.GetGoogleUserInfo(ctx, input)
		default:
			return nil, errors.New("invalid provider")
		}
	}()

	if err != nil {
		log.Printf("Error getting user info: %v", err)
		return nil, err
	}

	if userInfo.IsNewUser {
		defer func() {
			log.Printf("New user registered: %s, sending notification", userInfo.User.ProviderID)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			userRepo := users.NewRepository(ctx)

			ids, err := userRepo.GetTelegramIdsByNotificationEvent(users.NEW_USER)
			if err != nil {
				log.Printf("Error getting telegram ids: %v", err)
				return
			}

			service := telegram.NewService(ctx)

			msg := fmt.Sprintf("New User: %s %s %s", userInfo.User.Provider, userInfo.User.Email, userInfo.User.Name)

			err = service.SendBulkMessage(msg, ids)
			if err != nil {
				log.Printf("Error sending bulk message: %v", err)
				return
			}

			log.Printf("Notification sent to %d users", len(ids))

		}()
	}

	return &login.UserSubject{
		UserId:      userInfo.User.ID,
		WorkspaceID: userInfo.WorkspaceID,
	}, err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}
