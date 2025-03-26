package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

type UserInfo struct {
	UserId      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

func handler(ctx context.Context, input Input) (*UserInfo, error) {

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

	return userInfo, err
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
