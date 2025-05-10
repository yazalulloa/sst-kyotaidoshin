package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"io"
	"kyo-repo/internal/db/gen/model"
	"kyo-repo/internal/users"
	"kyo-repo/internal/util"
	"log"
	"net/http"
	"sync"
	"time"
)

const githubUrl = "https://api.github.com"

// Plan represents the user's plan details.
type Plan struct {
	Collaborators int    `json:"collaborators"`
	Name          string `json:"name"`
	Space         int    `json:"space"`
	PrivateRepos  int    `json:"private_repos"`
}

// PrivateUser represents a private user in the system.
type PrivateUser struct {
	Login                   string    `json:"login"`
	ID                      int64     `json:"id"`
	UserViewType            string    `json:"user_view_type"`
	NodeID                  string    `json:"node_id"`
	AvatarURL               string    `json:"avatar_url"`
	GravatarID              *string   `json:"gravatar_id,omitempty"`
	URL                     string    `json:"url"`
	HTMLURL                 string    `json:"html_url"`
	FollowersURL            string    `json:"followers_url"`
	FollowingURL            string    `json:"following_url"`
	GistsURL                string    `json:"gists_url"`
	StarredURL              string    `json:"starred_url"`
	SubscriptionsURL        string    `json:"subscriptions_url"`
	OrganizationsURL        string    `json:"organizations_url"`
	ReposURL                string    `json:"repos_url"`
	EventsURL               string    `json:"events_url"`
	ReceivedEventsURL       string    `json:"received_events_url"`
	Type                    string    `json:"type"`
	SiteAdmin               bool      `json:"site_admin"`
	Name                    *string   `json:"name,omitempty"`
	Company                 *string   `json:"company,omitempty"`
	Blog                    *string   `json:"blog,omitempty"`
	Location                *string   `json:"location,omitempty"`
	Email                   *string   `json:"email,omitempty"`
	NotificationEmail       *string   `json:"notification_email,omitempty"`
	Hireable                *bool     `json:"hireable,omitempty"`
	Bio                     *string   `json:"bio,omitempty"`
	TwitterUsername         *string   `json:"twitter_username,omitempty"`
	PublicRepos             int       `json:"public_repos"`
	PublicGists             int       `json:"public_gists"`
	Followers               int       `json:"followers"`
	Following               int       `json:"following"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	PrivateGists            int       `json:"private_gists"`
	TotalPrivateRepos       int       `json:"total_private_repos"`
	OwnedPrivateRepos       int       `json:"owned_private_repos"`
	DiskUsage               int       `json:"disk_usage"`
	Collaborators           int       `json:"collaborators"`
	TwoFactorAuthentication bool      `json:"two_factor_authentication"`
	Plan                    Plan      `json:"plan"`
	BusinessPlus            bool      `json:"business_plus"`
	LdapDN                  string    `json:"ldap_dn"`
}

// PublicUser represents a public user in the system.
type PublicUser struct {
	Login             string    `json:"login"`
	ID                int64     `json:"id"`
	UserViewType      string    `json:"user_view_type"`
	NodeID            string    `json:"node_id"`
	AvatarURL         string    `json:"avatar_url"`
	GravatarID        *string   `json:"gravatar_id,omitempty"`
	URL               string    `json:"url"`
	HTMLURL           string    `json:"html_url"`
	FollowersURL      string    `json:"followers_url"`
	FollowingURL      string    `json:"following_url"`
	GistsURL          string    `json:"gists_url"`
	StarredURL        string    `json:"starred_url"`
	SubscriptionsURL  string    `json:"subscriptions_url"`
	OrganizationsURL  string    `json:"organizations_url"`
	ReposURL          string    `json:"repos_url"`
	EventsURL         string    `json:"events_url"`
	ReceivedEventsURL string    `json:"received_events_url"`
	Type              string    `json:"type"`
	SiteAdmin         bool      `json:"site_admin"`
	Name              *string   `json:"name,omitempty"`
	Company           *string   `json:"company,omitempty"`
	Blog              *string   `json:"blog,omitempty"`
	Location          *string   `json:"location,omitempty"`
	Email             *string   `json:"email,omitempty"`
	NotificationEmail *string   `json:"notification_email,omitempty"`
	Hireable          *bool     `json:"hireable,omitempty"`
	Bio               *string   `json:"bio,omitempty"`
	TwitterUsername   *string   `json:"twitter_username,omitempty"`
	PublicRepos       int       `json:"public_repos"`
	PublicGists       int       `json:"public_gists"`
	Followers         int       `json:"followers"`
	Following         int       `json:"following"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Plan              Plan      `json:"plan"`
	PrivateGists      int       `json:"private_gists"`
	TotalPrivateRepos int       `json:"total_private_repos"`
	OwnedPrivateRepos int       `json:"owned_private_repos"`
	DiskUsage         int       `json:"disk_usage"`
	Collaborators     int       `json:"collaborators"`
}

type Email struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

func githubUserInfo(ctx context.Context, input Input) (*UserInfo, error) {
	httpClient := util.GetHttpClient()

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	var publicUser PublicUser
	emails := make([]Email, 0)

	header := http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", input.Tokenset.Raw.AccessToken)},
		"Accept":        []string{"application/vnd.github.v3+json"},
		//"Accept":               []string{"application/vnd.github.v3+json"},
		//"X-GitHub-Api-Version": []string{"2022-11-28"},
	}

	go func() {
		defer wg.Done()

		req, err := http.NewRequest("GET", githubUrl+"/user", nil)
		if err != nil {
			errorChan <- err
			return
		}
		req.Header = header

		res, err := httpClient.Do(req)

		if err != nil {
			errorChan <- err
			return
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		if res.StatusCode != http.StatusOK {

			byteArray, err := io.ReadAll(res.Body)
			if err != nil {
				errorChan <- err
				return
			}

			errorChan <- fmt.Errorf("failed to get publicUser: %d %s", res.StatusCode, string(byteArray))
			return
		}

		err = json.NewDecoder(res.Body).Decode(&publicUser)
		if err != nil {
			errorChan <- err
		}

		validate, err := util.GetValidator()
		if err != nil {
			errorChan <- err
			return
		}

		err = validate.Struct(publicUser)
		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			for _, valErr := range errors {
				log.Printf("Validation error: %v", valErr)
			}
			errorChan <- fmt.Errorf("validation error: %s", errors)
			return
		}
	}()

	go func() {
		defer wg.Done()

		req, err := http.NewRequest("GET", githubUrl+"/user/emails", nil)
		if err != nil {
			errorChan <- err
			return
		}
		req.Header = header

		res, err := httpClient.Do(req)

		if err != nil {
			errorChan <- err
			return
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		if res.StatusCode != http.StatusOK {

			byteArray, err := io.ReadAll(res.Body)
			if err != nil {
				errorChan <- err
				return
			}

			log.Printf("failed to get emails: %d %s", res.StatusCode, string(byteArray))
			return
		} else {

			err = json.NewDecoder(res.Body).Decode(&emails)

			if err != nil {
				errorChan <- err
			}
		}
	}()

	wg.Wait()
	close(errorChan)

	log.Printf("Errors: %d", len(errorChan))
	log.Printf("User: %+v", publicUser)
	log.Printf("Emails: %+v", emails)
	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	var primaryEmail string

	for _, email := range emails {
		if email.Primary {
			primaryEmail = email.Email
			break
		}
	}

	log.Printf("Primary Email: %s", primaryEmail)

	providerId := fmt.Sprint(publicUser.ID)

	userRepo := users.NewRepository(ctx)

	user, err := userRepo.GetByProvider(users.GITHUB, providerId)
	if err != nil {
		return nil, err
	}

	jsonByte, err := json.Marshal(publicUser)
	if err != nil {
		return nil, err
	}

	newUser := model.Users{
		ProviderID: providerId,
		Provider:   users.GITHUB.Name(),
		Email:      primaryEmail,
		Username:   *publicUser.Name,
		Name:       *publicUser.Name,
		Picture:    publicUser.AvatarURL,
		Data:       string(jsonByte),
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
		isNewUser:   true,
		WorkspaceID: "workspace-" + uuid.NewString(),
	}, nil
}
