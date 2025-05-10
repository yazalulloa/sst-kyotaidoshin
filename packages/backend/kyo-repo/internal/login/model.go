package login

import "github.com/yaz/kyo-repo/internal/db/gen/model"

type UserSubject struct {
	UserId      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

type UserInfo struct {
	User        *model.Users
	WorkspaceID string
	IsNewUser   bool
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
