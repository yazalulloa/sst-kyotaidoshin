package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"kyotaidoshin/util"
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

func githubUserInfo(input Input) (*UserInfo, error) {
	httpClient := &http.Client{
		Timeout: time.Minute * 7,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	var user PublicUser
	var emails []Email

	header := http.Header{
		"Authorization":        []string{fmt.Sprintf("Bearer %s", input.Tokenset.Raw.AccessToken)},
		"Accept":               []string{"application/vnd.github.v3+json"},
		"X-GitHub-Api-Version": []string{"2022-11-28"},
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

			errorChan <- fmt.Errorf("failed to get user: %d %s", res.StatusCode, string(byteArray))
			return
		}

		err = json.NewDecoder(res.Body).Decode(&user)
		if err != nil {
			errorChan <- err
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

			errorChan <- fmt.Errorf("failed to get emails: %d %s", res.StatusCode, string(byteArray))
			return
		}

		err = json.NewDecoder(res.Body).Decode(&emails)

		if err != nil {
			errorChan <- err
		}
	}()

	wg.Wait()
	close(errorChan)

	log.Printf("Errors: %d", len(errorChan))
	log.Printf("User: %+v", user)
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

	return &UserInfo{
		UserId:      "user-" + uuid.NewString(),
		WorkspaceID: "workspace-" + uuid.NewString(),
	}, nil
}
