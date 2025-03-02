package main

import (
	"context"
	"db/gen/model"
	"encoding/json"
	"github.com/google/uuid"
	xoauth "golang.org/x/oauth2"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"kyotaidoshin/users"
)

func googleUserInfo(ctx context.Context, input Input) (*UserInfo, error) {

	service, err := oauth2.NewService(ctx, option.WithTokenSource(xoauth.StaticTokenSource(&xoauth.Token{AccessToken: input.Tokenset.Access})))
	if err != nil {
		return nil, err
	}

	userinfo, err := service.Userinfo.V2.Me.Get().Do()
	if err != nil {
		return nil, err
	}

	providerId := userinfo.Id

	user, err := users.GetByProvider(users.GOOGLE, providerId)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return &UserInfo{
			UserId:      user.ID,
			WorkspaceID: "workspace-" + uuid.NewString(),
		}, nil
	}

	jsonByte, err := json.Marshal(userinfo)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	newId := id.String()
	newUser := model.Users{
		ID:         newId,
		ProviderID: providerId,
		Provider:   users.GOOGLE.Name(),
		Email:      userinfo.Email,
		Username:   userinfo.Name,
		Name:       userinfo.Name,
		Picture:    userinfo.Picture,
		Data:       string(jsonByte),
	}

	_, err = users.Insert(newUser)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		UserId:      newId,
		WorkspaceID: "workspace-" + uuid.NewString(),
	}, nil
}
