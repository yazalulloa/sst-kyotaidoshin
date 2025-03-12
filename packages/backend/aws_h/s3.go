package aws_h

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

		var notFound *types.NoSuchKey
		if ok := errors.As(err, &notFound); ok {
			fmt.Printf("Object %s does not exist in bucket %s\n", objectKey, bucketName)
		}

		is404 := strings.Contains(err.Error(), "response error StatusCode: 404")
		if !is404 {
			return false, err
		}

		return false, nil
	}

	return true, nil
}
