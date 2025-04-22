package util

import (
	"net/http"
	"sync"
	"time"
)

var httpClientInstance *http.Client
var httpClientOnce sync.Once

func GetHttpClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClientInstance = &http.Client{
			Timeout: time.Second * 30,
		}
	})
	return httpClientInstance
}
