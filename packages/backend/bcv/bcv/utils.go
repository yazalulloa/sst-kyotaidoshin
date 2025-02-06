package bcv

import "github.com/sst/sst/v3/sdk/golang/resource"

func GetBcvBucket() (string, error) {
	bucketName, err := resource.Get("bcv-bucket", "name")
	if err != nil {
		return "", err
	}

	return bucketName.(string), nil
}
