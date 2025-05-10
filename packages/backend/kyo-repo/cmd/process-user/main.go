package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"kyo-repo/internal/db/gen/model"
	"kyo-repo/internal/telegram"
	"kyo-repo/internal/users"
	"log"
	"time"
)

type UserSubject struct {
	UserId      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

type UserInfo struct {
	User        *model.Users
	WorkspaceID string
	isNewUser   bool
}

func handler(ctx context.Context, input Input) (*UserSubject, error) {

	userInfo, err := func() (*UserInfo, error) {
		switch input.Provider {
		case "github":
			return githubUserInfo(ctx, input)
		case "google":
			return googleUserInfo(ctx, input)
		default:
			return nil, errors.New("invalid provider")
		}
	}()

	if err != nil {
		log.Printf("Error getting user info: %v", err)
		return nil, err
	}

	if userInfo.isNewUser {
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

	return &UserSubject{
		UserId:      userInfo.User.ID,
		WorkspaceID: userInfo.WorkspaceID,
	}, err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(handler)
}

type Input struct {
	Provider string `json:"provider"`
	ClientID string `json:"clientID"`
	Tokenset struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
		Expiry  int    `json:"expiry"`
		Raw     struct {
			AccessToken           string `json:"access_token"`
			ExpiresIn             int    `json:"expires_in"`
			RefreshToken          string `json:"refresh_token"`
			RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
			TokenType             string `json:"token_type"`
			Scope                 string `json:"scope"`
			IdToken               string `json:"id_token"`
		} `json:"raw"`
	} `json:"tokenset"`
}
