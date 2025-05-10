package login

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/users"
	xoauth "golang.org/x/oauth2"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

func GetGoogleUserInfo(ctx context.Context, input Input) (*UserInfo, error) {

	service, err := oauth2.NewService(ctx, option.WithTokenSource(xoauth.StaticTokenSource(&xoauth.Token{AccessToken: input.Tokenset.Access})))
	if err != nil {
		return nil, err
	}

	userinfo, err := service.Userinfo.V2.Me.Get().Do()
	if err != nil {
		return nil, err
	}

	providerId := userinfo.Id

	jsonByte, err := json.Marshal(userinfo)
	if err != nil {
		return nil, err
	}

	newUser := model.Users{
		ProviderID: providerId,
		Provider:   users.GOOGLE.Name(),
		Email:      userinfo.Email,
		Username:   userinfo.Name,
		Name:       userinfo.Name,
		Picture:    userinfo.Picture,
		Data:       string(jsonByte),
	}

	userRepo := users.NewRepository(ctx)

	user, err := userRepo.GetByProvider(users.GOOGLE, providerId)
	if err != nil {
		return nil, err
	}

	if user != nil {
		newUser.ID = user.ID

		_, err = userRepo.UpdateWithLogin(newUser)
		if err != nil {
			return nil, err
		}

		return &UserInfo{
			User:        user,
			WorkspaceID: "workspace-" + uuid.NewString(),
		}, nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	newId := id.String()
	newUser.ID = newId

	_, err = userRepo.Insert(newUser)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		User:        &newUser,
		IsNewUser:   true,
		WorkspaceID: "workspace-" + uuid.NewString(),
	}, nil
}
