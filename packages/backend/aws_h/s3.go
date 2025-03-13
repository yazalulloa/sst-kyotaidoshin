package aws_h

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"strings"
)

func FileExistsS3(ctx context.Context, bucketName string, objectKey string) (bool, error) {

	s3Client, err := GetS3Client(ctx)
	if err != nil {
		return false, err
	}

	_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	})

	if err != nil {

		is404 := strings.Contains(err.Error(), "response error StatusCode: 404")
		if !is404 {
			log.Printf("error HeadObject %s %s %s", objectKey, bucketName, err)
			return false, err
		}

		return false, nil
	}

	return true, nil
}
