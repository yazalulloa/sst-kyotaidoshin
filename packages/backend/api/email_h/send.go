package email_h

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/wneessen/go-mail"
	"sync"
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

func SendEmail(ctx context.Context, emailKey string, msg *mail.Msg) error {

	configMap, err := loadConfigMap()
	if err != nil {
		return err
	}

	config, ok := configMap[emailKey]
	if !ok {
		return fmt.Errorf("email key %s not found", emailKey)
	}

	if err := msg.From(config.Username); err != nil {
		return err
	}

	client, err := mail.NewClient(config.Host, mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(config.Username), mail.WithPassword(config.Password))

	if err != nil {
		return fmt.Errorf("failed to create mail client: %v", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}

	return nil
}

//func SendEmail(ctx context.Context, emailKey string, msg *mail.Msg) error {
//
//	jsonStr, err := base64.URLEncoding.DecodeString(encoded)
//
//	if err != nil {
//		return err
//	}
//
//	var configs []MailerConfig
//	err = json.Unmarshal(jsonStr, &configs)
//	if err != nil {
//		return err
//	}
//
//	config := configs[0]
//	log.Printf("Sending email with config: %v", config)
//
//	message := mail.NewMsg()
//	if err := message.From(config.Username); err != nil {
//		return err
//	}
//	if err := message.To("yzlup2@gmail.com"); err != nil {
//		return err
//	}
//
//	message.Subject("This is my first mail with go-mail!")
//	message.SetBodyString(mail.TypeTextPlain, "Do you like this mail? I certainly do!")
//
//	client, err := mail.NewClient(config.Host, mail.WithTLSPortPolicy(mail.TLSMandatory),
//		mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(config.Username), mail.WithPassword(config.Password))
//	//authMethod := mail.SMTPAuthLogin
//	//
//	//log.Printf("With auth method: %v", authMethod)
//	//
//	//client, err := mail.NewClient(config.Host, mail.WithSMTPAuth(authMethod),
//	//	mail.WithPort(config.Port), mail.WithUsername(config.Username), mail.WithPassword(config.Password))
//	if err != nil {
//		log.Printf("Failed to create mail client: %v", err)
//		return fmt.Errorf("failed to create mail client: %v", err)
//	}
//
//	if err := client.DialAndSendWithContext(ctx, message); err != nil {
//		log.Printf("Failed to send mail: %v", err)
//		return fmt.Errorf("failed to send mail: %s", err)
//	}
//
//	return nil
//}
