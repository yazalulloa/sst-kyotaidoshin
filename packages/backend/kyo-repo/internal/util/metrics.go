package util

import (
	"errors"
	"github.com/posthog/posthog-go"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"log"
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

type Exception struct {
	Type      *string    `json:"type"`
	Value     *string    `json:"value"`
	Mechanism *Mechanism `json:"mechanism"`
	Module    *string    `json:"module"`
	ThreadId  *int64     `json:"thread_id"`
	//	stacktrace?: {
	//	frames?: StackFrame[]
	//	type: 'raw'
	//}
}

type Mechanism struct {
	Handled   *bool   `json:"handled"`
	Type      *string `json:"type"`
	Source    *string `json:"source"`
	Synthetic *bool   `json:"synthetic"`
}

type StackFrame struct {
	Platform        string          `json:"platform"`
	Filename        *string         `json:"filename,omitempty"`
	Function        *string         `json:"function,omitempty"`
	Module          *string         `json:"module,omitempty"`
	Lineno          *int            `json:"lineno,omitempty"`
	Colno           *int            `json:"colno,omitempty"`
	AbsPath         *string         `json:"abs_path,omitempty"`
	ContextLine     *string         `json:"context_line,omitempty"`
	PreContext      *[]string       `json:"pre_context,omitempty"`
	PostContext     *[]string       `json:"post_context,omitempty"`
	InApp           *bool           `json:"in_app,omitempty"`
	InstructionAddr *string         `json:"instruction_addr,omitempty"`
	AddrMode        *string         `json:"addr_mode,omitempty"`
	Vars            *map[string]any `json:"vars,omitempty"`
	ChunkId         *string         `json:"chunk_id,omitempty"`
}

func PosthogException(userId string, errList []string) {
	client, err := GetPosthogClient()
	if err != nil {
		log.Println("PosthogException: failed to get Posthog client:", err)
		return
	}

	if client == nil {
		log.Println("PosthogException: Posthog client is nil")
		return
	}

	c := *client

	c.Enqueue(posthog.Capture{
		Event:      "$exception",
		DistinctId: userId,
		Properties: posthog.NewProperties().
			Set("exception_list", errList),
	})

}
