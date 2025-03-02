package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"kyotaidoshin/users"
)

type UserInfo struct {
	UserId      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

func handler(ctx context.Context, input Input) (*UserInfo, error) {

	userInfo, err := func() (*UserInfo, error) {
		switch input.Provider {
		case "github":
			return githubUserInfo(input)
		case "google":
			return googleUserInfo(ctx, input)
		default:
			return nil, errors.New("invalid provider")
		}
	}()

	if err == nil && userInfo != nil {
		_, err = users.UpdateLastLogin(userInfo.UserId)
		if err != nil {
			return nil, err
		}
	}

	return userInfo, err
}

func main() {
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
