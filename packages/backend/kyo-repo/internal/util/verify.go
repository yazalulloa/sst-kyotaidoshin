package util

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"log"
)

const USER_ID = "userID"
const USER_PAYLOAD_KEY = "userPayload"

type UserPayload struct {
	Type       string `json:"type"`
	Properties struct {
		UserID      string `json:"userID"`
		WorkspaceID string `json:"workspaceID"`
	} `json:"properties"`
}

func Verify(ctx context.Context, accessToken, refreshToken string) (context.Context, error) {
	//timestamp := time.Now().UnixMilli()
	//defer func() { log.Printf("Verify took %d ms", time.Now().UnixMilli()-timestamp) }()

	verifyAccessFunction, err := resource.Get("VerifyAccess", "name")
	if err != nil {
		panic(fmt.Errorf("error getting VerifyAccess function: %v", err))
	}

	client, err := aws_h.GetLambdaClient(ctx)
	if err != nil {
		return nil, err
	}

	output, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(verifyAccessFunction.(string)),
		Payload:        []byte(`{"accessToken":"` + accessToken + `","refreshToken":"` + refreshToken + `"}`),
		InvocationType: types.InvocationTypeRequestResponse,
	})

	if err != nil {
		log.Printf("Error invoking VerifyAccess lambda: %v", err)
		return nil, err
	}
	payload := string(output.Payload)
	if output.FunctionError != nil {
		log.Printf("Function error: %v %s", *output.FunctionError, payload)
		return nil, fmt.Errorf("function error: %s", payload)
	}

	//log.Printf("User payload: %s", payload)

	var userPayload UserPayload
	err = json.Unmarshal(output.Payload, &userPayload)
	if err != nil {
		log.Printf("Error unmarshalling payload: %v", err)
		return nil, err
	}

	newCtx := context.WithValue(ctx, USER_PAYLOAD_KEY, payload)
	newCtx = context.WithValue(newCtx, USER_ID, userPayload.Properties.UserID)

	return newCtx, nil
}
