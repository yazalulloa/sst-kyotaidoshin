package util

import (
	"aws_h"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CsrfKey = "csrf-key"

// const CsrfInputName = "kyotaidogo-csrf-form"
const CsrfInputName = "gorilla.csrf.Token"

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

func GetQueryParamAsSortOrderType(r *http.Request, paramName string) SortOrderType {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return SortOrderTypeDESC
	}

	param = strings.ToUpper(param)

	if param == "ASC" {
		return SortOrderTypeASC
	}

	return SortOrderTypeDESC
}

func UuidV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	return id.String()
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

func MonthsToInt(months string) []int16 {
	return StrArrayToInt16Array(strings.Split(months, ","))
}

func Int16ArrayToString(int16Array []int16) string {
	strArray := make([]string, len(int16Array))
	for i, v := range int16Array {
		strArray[i] = strconv.Itoa(int(v))
	}
	return strings.Join(strArray, ",")
}

func StrArrayToInt16Array(strArray []string) []int16 {
	int16Array := make([]int16, len(strArray))
	for i, v := range strArray {
		int16Array[i] = StringToInt16(v)
	}
	return int16Array
}

func GetUploadFormParams(r *http.Request, filePrefix string) (*UploadBackupParams, error) {

	filename := GetQueryParamAsString(r, "name")

	if filename == "" {
		return nil, errors.New("BAD REQUEST")
	}

	bucketName, err := GetReceiptsBucket()
	if err != nil {
		log.Printf("Error getting bucket Name: %s", err)
		return nil, err
	}

	//url := r.Header.Get("origin")
	//if url == "" {
	//	return nil, fmt.Errorf("origin header not found")
	//}

	//url := fmt.Sprintf("%s://%s/", r.URL.Scheme, r.URL.Host)
	//redirectUrl := fmt.Sprintf("%s/%s", url, uploadPath)
	metaUuid := uuid.New().String()

	conditions := []interface{}{
		//map[string]string{"success_action_redirect": redirectUrl},
		//[]interface{}{"starts-with", "$Content-Type", "application/gzip"},
		map[string]string{"x-amz-meta-uuid": metaUuid},
		map[string]string{"x-amz-meta-filename": filename},
		map[string]string{"success_action_status": "204"},
		//[]interface{}{"starts-with", "$x-amz-meta-tag", ""},
		[]interface{}{"content-length-range", 1, 2048576},
	}

	optionFn := func(options *s3.PresignPostOptions) {
		options.Expires = time.Minute
		options.Conditions = conditions
	}

	presignedPostRequest, err := aws_h.PresignPostObject(r.Context(), bucketName, fmt.Sprintf("%s_%s", filePrefix, uuid.New().String()), optionFn)
	if err != nil {
		return nil, err
	}

	//presignedPostRequest.Values["success_action_redirect"] = redirectUrl
	presignedPostRequest.Values["x-amz-meta-uuid"] = metaUuid
	presignedPostRequest.Values["success_action_status"] = "204"
	presignedPostRequest.Values["x-amz-meta-filename"] = filename

	return &UploadBackupParams{
		Url:    presignedPostRequest.URL,
		Values: presignedPostRequest.Values,
	}, nil
}

type UploadBackupParams struct {
	Url              string
	Values           map[string]string
	OutOfBandsUpdate bool
}

func StringToInt32(str string) int32 {
	value, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

func StringToInt64(str string) int64 {
	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return int64(value)
}
