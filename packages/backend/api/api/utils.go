package api

import (
	"aws_h"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"io"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CsrfKey = "csrf-key"

// const CsrfInputName = "kyotaidogo-csrf-form"
const CsrfInputName = "gorilla.csrf.Token"

var ErrNoRows = errors.New("qrm: no rows in result set")

func Encode(obj any) *string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(obj)
	if err != nil {
		panic(err)
	}

	encoded := base64.URLEncoding.EncodeToString(b.Bytes())
	return &encoded
}

func Decode(encoded string, obj any) error {
	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		log.Println(`failed base64 Decode`, err)
		return err
	}
	d := gob.NewDecoder(bytes.NewReader(b))
	err = d.Decode(obj)
	if err != nil {
		log.Println(`failed gob Decode`, err)
		return err
	}

	return nil
}

func GetQueryParamAsInt(r *http.Request, paramName string) int64 {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return 0
	}
	value, err := strconv.ParseInt(param, 10, 64)
	//value, err := strconv.Atoi(param)
	if err != nil {
		return 0
	}
	return value
}

func GetQueryParamAsDate(r *http.Request, paramName string) *time.Time {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return nil
	}

	value, err := time.Parse(time.DateOnly, param)
	if err != nil {
		return nil
	}

	return &value
}

func GetQueryParamAsTimestamp(r *http.Request, paramName string) *time.Time {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return nil
	}

	unix, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return nil
	}

	value := time.UnixMilli(unix)

	return &value
}

func GetQueryParamAsString(r *http.Request, paramName string) string {
	return strings.TrimSpace(r.URL.Query().Get(paramName))
}

func GetQueryParamAsSortOrderType(r *http.Request, paramName string) util.SortOrderType {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return util.SortOrderTypeDESC
	}

	param = strings.ToUpper(param)

	if param == "ASC" {
		return util.SortOrderTypeASC
	}

	return util.SortOrderTypeDESC
}

func CrsfHeaders(ctx context.Context) string {
	token := ctx.Value("gorilla.csrf.Token").(string)
	return fmt.Sprintf("{\"%s\":\"%s\"}", "X-CSRF-Token", token)
}

func ToASCII(str string) string {
	var builder strings.Builder
	for _, character := range str {
		builder.WriteString(fmt.Sprintf("%d", character))
	}
	return builder.String()
}

func PadLeft(s string, l int) string {
	if len(s) >= l {
		return s
	}
	return strings.Repeat("0", l-len(s)) + s
}

func IsDevMode() bool {
	return true
}

//func FileHash(filepath string) (int64, error) {
//	file, err := os.Open(filepath)
//
//	if err != nil {
//		return 0, err
//	}
//	defer func(file *os.File) {
//		err := file.Close()
//		if err != nil {
//			log.Println("Error closing file:", err)
//			return
//		}
//	}(file)
//
//	buf := make([]byte, 1024*1024)
//	hash := xxhash.New()
//	if _, err := io.CopyBuffer(hash, file, buf); err != nil {
//		return 0, err
//	}
//	bytesSum := hash.Sum(nil)
//	fileHash := int64(xxhash.Sum64(bytesSum))
//	return fileHash, nil
//}

//func Render(ctx echo.Context, statusCode int, t templ.Component) error {
//	buf := templ.GetBuffer()
//	defer templ.ReleaseBuffer(buf)
//	reqCtx := ctx.Request().Context()
//	reqCtx = context.WithValue(reqCtx, CsrfKey, ctx.Get(CsrfKey))
//	if err := t.Render(reqCtx, buf); err != nil {
//		return err
//	}
//
//	return ctx.HTML(statusCode, buf.String())
//}

func BuildUploadForm(ctx context.Context, uploadPath string, filePrefix string) (templ.Component, error) {
	bucketName, err := resource.Get("UploadBackup", "name")
	if err != nil {
		log.Printf("Error getting bucket name: %s", err)
		return nil, err
	}

	functionUrl, err := resource.Get("ApiFunction", "url")
	if err != nil {
		log.Printf("Error getting function url: %s", err)
		return nil, err
	}

	redirectUrl := fmt.Sprintf("%s%s", functionUrl.(string), uploadPath)
	metaUuid := uuid.New().String()

	conditions := []interface{}{
		map[string]string{"success_action_redirect": redirectUrl},
		//[]interface{}{"starts-with", "$Content-Type", "application/gzip"},
		map[string]string{"x-amz-meta-uuid": metaUuid},
		//[]interface{}{"starts-with", "$x-amz-meta-tag", ""},
		[]interface{}{"content-length-range", 1, 2048576},
	}

	optionFn := func(options *s3.PresignPostOptions) {
		//options.Expires = time.Hour
		options.Conditions = conditions
	}

	presignedPostRequest, err := aws_h.PresignPostObject(ctx, bucketName.(string), fmt.Sprintf("%s_%s", filePrefix, uuid.New().String()), optionFn)
	if err != nil {
		return nil, err
	}

	presignedPostRequest.Values["success_action_redirect"] = redirectUrl
	presignedPostRequest.Values["x-amz-meta-uuid"] = metaUuid

	return UploadBackupForm(presignedPostRequest.URL, presignedPostRequest.Values), nil
}

func ProcessUploadBackup(r *http.Request, uploadBackupFormUrl string, idUpdater string, event string, processJson func(*json.Decoder) (int64, error)) (templ.Component, error) {
	bucket := GetQueryParamAsString(r, "bucket")
	key := GetQueryParamAsString(r, "key")
	etag := GetQueryParamAsString(r, "etag")

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

	_, err = s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		log.Printf("Error deleting object from bucket %s key %s: %s", bucket, key, err)
	}

	return UploadBackupResponse(inserted, uploadBackupFormUrl, idUpdater, event), nil
}
