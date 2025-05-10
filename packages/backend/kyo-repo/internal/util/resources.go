package util

import (
	"fmt"
	"github.com/sst/sst/v3/sdk/golang/resource"
)

func GetReceiptsBucket() (string, error) {

	bucketName, err := resource.Get("ReceiptsBucket", "name")
	if err != nil {
		return "", fmt.Errorf("error getting receipts bucket name: %v", err)
	}

	return bucketName.(string), nil
}
