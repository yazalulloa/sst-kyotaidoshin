package api

import (
	"aws_h"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"kyotaidoshin/util"
	"log"
	"net/http"
)

func BuildUploadForm(ctx context.Context, uploadPath string, filePrefix string) (templ.Component, error) {
	params, err := util.GetUploadFormParams(ctx, uploadPath, filePrefix)
	if err != nil {
		return nil, err
	}
	return UploadBackupForm(*params), nil
}

func ProcessUploadBackup(r *http.Request, uploadBackupFormUrl string, idUpdater string, event string, processJson func(*json.Decoder) (int64, error)) (templ.Component, error) {
	bucket := util.GetQueryParamAsString(r, "bucket")
	key := util.GetQueryParamAsString(r, "key")
	etag := util.GetQueryParamAsString(r, "etag")

	if bucket == "" || key == "" || etag == "" {
		return nil, errors.New("Bad Request")
	}

	s3Client, err := aws_h.GetS3Client(r.Context())
	if err != nil {
		log.Printf("Error getting s3 client: %s", err)
		return nil, err
	}

	log.Printf("Getting object from bucket %s key %s", bucket, key)
	outPut, err := s3Client.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket:       &bucket,
		Key:          &key,
		IfMatch:      &etag,
		ChecksumMode: types.ChecksumModeEnabled,
	})

	if err != nil {
		log.Printf("Error getting object from bucket %s key %s: %s", bucket, key, err)
		return nil, err
	}

	deleteObj := func() {
		log.Printf("Deleting object from bucket %s key %s", bucket, key)
		_, err = s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})

		if err != nil {
			log.Printf("Error deleting object from bucket %s key %s: %s", bucket, key, err)
		}
	}

	defer deleteObj()

	closeBody := func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing: ", err)
			return
		}
	}

	defer closeBody(outPut.Body)

	gzipReader, err := gzip.NewReader(outPut.Body)
	if err != nil {
		log.Printf("Error creating gzip reader: %s", err)
		return nil, err
	}

	defer closeBody(gzipReader)

	decoder := json.NewDecoder(gzipReader)

	var inserted int64 = 0
	//apts := make([]ApartmentDto, 50)

	for decoder.More() { // Loop through JSON objects in the stream

		rowsAffected, err := processJson(decoder)
		if err != nil {
			return nil, err
		}
		inserted += rowsAffected
	}

	return UploadBackupResponse(inserted, uploadBackupFormUrl, idUpdater, event), nil
}
