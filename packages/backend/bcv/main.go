package main

import (
	"bcv/bcv"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"time"
)

func bcvTask() (string, error) {

	timestamp := time.Now().UnixMilli()

	if true {
		err := bcv.Check()
		if err != nil {
			return "", err
		}

		return "OK", nil
	}

	links, err := bcv.AllFiles()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(links)
	if err != nil {
		return "", err
	}

	log.Printf("bcvTask took %d ms", time.Now().UnixMilli()-timestamp)
	return string(jsonBytes), nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(bcvTask)
}
