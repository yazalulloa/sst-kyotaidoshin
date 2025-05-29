package util

import (
	"errors"
	"github.com/posthog/posthog-go"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"sync"
)

var posthogClientInstance *posthog.Client
var posthogClientOnce sync.Once

func GetPosthogClient() (*posthog.Client, error) {
	var oErr error
	posthogClientOnce.Do(func() {
		apiKey, err := resource.Get("PosthogApiKey", "value")
		if err != nil {
			oErr = errors.Join(err, errors.New("failed to get posthog API key"))
			return
		}
		if apiKey == "" {
			oErr = errors.New("posthog API key is not set")
			return
		}

		client, err := posthog.NewWithConfig(apiKey.(string), posthog.Config{Endpoint: "https://us.i.posthog.com"})
		if err != nil {
			oErr = err
		} else {
			posthogClientInstance = &client
		}

	})

	return posthogClientInstance, oErr
}
