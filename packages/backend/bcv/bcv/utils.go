package bcv

import "github.com/sst/sst/v3/sdk/golang/resource"

func GetBcvBucket() (string, error) {
	bucketName, err := resource.Get("bcv-bucket", "name")
	if err != nil {
		return "", err
	}

	return bucketName.(string), nil
}

func GetParsingLambda() (string, error) {
	lambdaName, err := resource.Get("BcvFileParser", "name")
	if err != nil {
		return "", err
	}

	return lambdaName.(string), nil
}
