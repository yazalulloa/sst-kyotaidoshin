package email_h

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/wneessen/go-mail"
	"log"
	"os"
	"sync"
	"time"
)

var configMap map[string]MailerConfig
var configOnce sync.Once

type MailerConfig struct {
	Key      string `json:"key"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func GetConfigs() (map[string]MailerConfig, error) {
	return loadConfigMap()
}

func loadConfigMap() (map[string]MailerConfig, error) {

	var outErr error

	configOnce.Do(func() {
		configStr, err := resource.Get("MailerConfigs", "value")

		if err != nil {
			outErr = err
			return
		}

		jsonStr, err := base64.URLEncoding.DecodeString(configStr.(string))

		if err != nil {
			outErr = err
			return
		}

		var configs []MailerConfig
		err = json.Unmarshal(jsonStr, &configs)
		if err != nil {
			outErr = err
			return
		}

		configMap = make(map[string]MailerConfig, len(configs))
		for _, config := range configs {
			configMap[config.Key] = config
		}

	})

	return configMap, outErr
}

type MsgWithCallBack struct {
	Msg            *mail.Msg
	Callback       func()
	ShouldContinue func() bool
}

func GetFromEmail(emailKey string) (string, error) {

	configMap, err := loadConfigMap()
	if err != nil {
		return "", err
	}

	config, ok := configMap[emailKey]
	if !ok {
		return "", fmt.Errorf("email key %s not found", emailKey)
	}

	return config.Username, nil
}

func SendEmail(ctx context.Context, emailKey string, messages []*MsgWithCallBack) error {

	configMap, err := loadConfigMap()
	if err != nil {
		return err
	}

	config, ok := configMap[emailKey]
	if !ok {
		return fmt.Errorf("email key %s not found", emailKey)
	}

	for _, msg := range messages {

		if err := msg.Msg.From(config.Username); err != nil {
			return err
		}
	}

	client, err := mail.NewClient(config.Host, mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(config.Username), mail.WithPassword(config.Password))

	if err != nil {
		return fmt.Errorf("failed to create mail client: %v", err)
	}

	if err := clientSend(client, ctx, messages); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}

	return nil
}

func clientSend(c *mail.Client, ctx context.Context, messages []*MsgWithCallBack) error {
	client, err := c.DialToSMTPClientWithContext(ctx)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer func() {
		err = c.CloseWithSMTPClient(client)
		if err != nil {
			log.Printf("failed to close connection: %s", err)
		}
	}()

	for _, msg := range messages {

		if !msg.ShouldContinue() {
			log.Printf("Cancelled")
			break
		}

		if os.Getenv("SEND_MAIL") == "true" {
			if err = c.SendWithSMTPClient(client, msg.Msg); err != nil {
				return fmt.Errorf("send failed: %w", err)
			}
		} else {
			time.Sleep(1 * time.Second)
		}

		msg.Callback()
	}

	return nil
}
