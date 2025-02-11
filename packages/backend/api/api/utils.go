package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
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

func Base64Encode(obj any) *string {
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
