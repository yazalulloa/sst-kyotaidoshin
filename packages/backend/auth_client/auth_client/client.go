package auth_client

import (
	"encoding/json"
	"fmt"
	"github.com/ROU-Technology/openauth-go"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"log"
	"regexp"
	"sync"
)

var once sync.Once
var instance *Holder

type Holder struct {
	Subjects openauth.SubjectSchema
	Client   *openauth.Client
}

func GetClient() (*Holder, error) {

	var err error
	once.Do(func() {
		url, nerr := resource.Get("AuthServer", "url")
		if nerr != nil {
			log.Printf("Error getting AuthServer url: %s", err)
			err = nerr
			return
		}

		clientId, nerr := resource.Get("AppClientId", "value")
		if nerr != nil {
			log.Printf("Error getting MyAuth url: %s", err)
			err = nerr
			return
		}

		instance = &Holder{
			Client: openauth.NewClient(clientId.(string), url.(string)),
			Subjects: openauth.SubjectSchema{
				"user": func(props interface{}) error {
					// Type assert to map
					properties, ok := props.(map[string]interface{})
					if !ok {
						return fmt.Errorf("expected map[string]interface{}, got %T", props)
					}

					// Check if email exists
					email, ok := properties["email"].(string)
					if !ok {
						return fmt.Errorf("email is required and must be a string")
					}

					// Validate email format
					emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
					if !emailRegex.MatchString(email) {
						return fmt.Errorf("invalid email format")
					}

					return nil
				},
			},
		}
	})

	return instance, err
}

func Verify(accessToken, refreshToken string) error {
	holder, err := GetClient()
	if err != nil {
		return err
	}

	var options *openauth.VerifyOptions
	if refreshToken != "" {
		options = &openauth.VerifyOptions{
			RefreshToken: refreshToken,
		}
	}

	subject, err := holder.Client.Verify(holder.Subjects, accessToken, options)
	if err != nil {
		log.Printf("Failed to verify token: %v", err)
		return err
	}

	bytes, _ := json.Marshal(subject)
	log.Printf("Subject: %s", string(bytes))

	return nil
}
