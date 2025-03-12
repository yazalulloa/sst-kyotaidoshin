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

func BuildUploadForm(r *http.Request, filePrefix string) (templ.Component, error) {

	params, err := util.GetUploadFormParams(r, filePrefix)
	if err != nil {
		return nil, err
	}
	return UploadFormView(*params), nil
}

func ProcessBackup(ctx context.Context, bucket, key, etag *string,
	processJson func(*json.Decoder) (int64, error)) (int64, error) {
	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		log.Printf("Error getting s3 client: %s", err)
		return 0, err
	}

	log.Printf("Getting object from bucket %s key %s", *bucket, *key)
	outPut, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:       bucket,
		Key:          key,
		IfMatch:      etag,
		ChecksumMode: types.ChecksumModeEnabled,
	})

	if err != nil {
		log.Printf("Error getting object from bucket %s key %s: %s", *bucket, *key, err)
		return 0, err
	}

	deleteObj := func() {
		log.Printf("Deleting object from bucket %s key %s", *bucket, *key)
		_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: bucket,
			Key:    key,
		})

		if err != nil {
			log.Printf("Error deleting object from bucket %s key %s: %s", *bucket, *key, err)
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
		return 0, err
	}

	defer closeBody(gzipReader)

	decoder := json.NewDecoder(gzipReader)

	var inserted int64 = 0

	for decoder.More() {
		rowsAffected, err := processJson(decoder)
		if err != nil {
			return 0, err
		}
		inserted += rowsAffected
	}

	return inserted, nil
}

func ProcessUploadBackup(r *http.Request, redirecUrl string, processJson func(*json.Decoder) (int64, error)) (templ.Component, error) {
	key := r.FormValue("key")
	if key == "" {
		log.Printf("key is empty")
		return nil, errors.New("BAD REQUEST")
	}

	bucket, err := util.GetReceiptsBucket()
	if err != nil {
		log.Printf("Error getting bucket Name: %s", err)
		return nil, err
	}

	_, err = ProcessBackup(r.Context(), &bucket, &key, nil, processJson)
	if err != nil {
		return nil, err
	}

	return AnchorClickInitView(redirecUrl), nil
}

func OLD_ProcessUploadBackup(r *http.Request, uploadBackupFormUrl string, idUpdater string, event string, processJson func(*json.Decoder) (int64, error)) (templ.Component, error) {
	bucket := util.GetQueryParamAsString(r, "bucket")
	key := util.GetQueryParamAsString(r, "key")
	etag := util.GetQueryParamAsString(r, "etag")

	if bucket == "" || key == "" || etag == "" {
		return nil, errors.New("bad Request")
	}

	inserted, err := ProcessBackup(r.Context(), &bucket, &key, &etag, processJson)
	if err != nil {
		return nil, err
	}

	return UploadBackupResponse(inserted, uploadBackupFormUrl, idUpdater, event), nil
}
