package aws_h

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func PutFile(ctx context.Context, bucketName, objectKey, contentType string, filePath string) (interface{}, error) {
	s3Client, err := GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	contentLength := fileInfo.Size()
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(bucketName),
		Key:               aws.String(objectKey),
		Body:              file,
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		//ChecksumCRC32:             nil,
		//ChecksumCRC32C:            nil,
		//ChecksumSHA1:              nil,
		//ChecksumSHA256:            nil,
		ContentLength: &contentLength,
		ContentType:   aws.String(contentType),
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func PutBuffer(ctx context.Context, bucketName, objectKey, contentType string, buf *bytes.Buffer) (interface{}, error) {
	s3Client, err := GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	contentLength := int64(buf.Len())
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(bucketName),
		Key:               aws.String(objectKey),
		Body:              buf,
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		//ChecksumCRC32:             nil,
		//ChecksumCRC32C:            nil,
		//ChecksumSHA1:              nil,
		//ChecksumSHA256:            nil,
		ContentLength: &contentLength,
		ContentType:   aws.String(contentType),
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func GetObjectBuffer(ctx context.Context, bucketName, objectKey string) ([]byte, error) {
	s3Client, err := GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func WriteObjectToDisk(ctx context.Context, bucketName, objectKey, filePath string) error {
	s3Client, err := GetS3Client(ctx)
	if err != nil {
		return err
	}

	res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}
