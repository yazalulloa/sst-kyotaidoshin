package util

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var ErrNoRows = errors.New("qrm: no rows in result set")

const OneKb = 1024.0
const OneMb = 1024 * OneKb
const OneGb = 1024 * OneMb

type SortOrderType string

const (
	SortOrderTypeASC  SortOrderType = "ASC"
	SortOrderTypeDESC SortOrderType = "DESC"
)

func HtmlCurrencies() string {
	return StringArrayToString(AllowedCurrenciesStringArray())
}

func StringArrayToString(array []string) string {
	var b strings.Builder
	b.WriteString("[")
	for _, v := range array {
		b.WriteString("'")
		b.WriteString(v)
		b.WriteString("',")
	}
	b.WriteString("]")

	return b.String()
}

type UnitSizeTuple struct {
	name string
	size float64
}

var UnitSizeTuples = []UnitSizeTuple{
	{name: "GB", size: OneGb},
	{name: "MB", size: OneMb},
	{name: "KB", size: OneKb},
}

func FormatBytes(bytes int64) string {
	size := float64(bytes)
	for _, tuple := range UnitSizeTuples {
		if tuple.size < size {
			var quotient = size / tuple.size
			if quotient > 0 {
				formatted := fmt.Sprintf("%.4f", quotient)
				dotPos := strings.Index(formatted, ".")

				if dotPos != -1 {
					done := false
					for done == false {
						if last := len(formatted) - 1; last >= dotPos && formatted[last] == '0' {
							formatted = formatted[:last]
						} else {
							done = true
						}
					}
				}

				return fmt.Sprintf("%s %s", formatted, tuple.name)
			}
		}
	}

	return fmt.Sprintf("%.2f bytes", size)
}

func NotBlank(fl validator.FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.String:
		return len(strings.Trim(strings.TrimSpace(field.String()), "\x1c\x1d\x1e\x1f")) > 0
	case reflect.Chan, reflect.Map, reflect.Slice, reflect.Array:
		return field.Len() > 0
	case reflect.Ptr, reflect.Interface, reflect.Func:
		return !field.IsNil()
	default:
		return field.IsValid() && field.Interface() != reflect.Zero(field.Type()).Interface()
	}
}

var validatorInstance *validator.Validate
var once sync.Once

func GetValidator() (*validator.Validate, error) {
	var err error
	once.Do(func() {
		validatorInstance = validator.New(validator.WithRequiredStructEnabled())
		err = validatorInstance.RegisterValidation("notblank", NotBlank)
	})

	return validatorInstance, err
}

func SplitArray[T any](arr []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(arr); i += chunkSize {
		end := i + chunkSize
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

func StringToInt16(str string) int16 {
	num, err := strconv.ParseInt(str, 10, 16)
	if err != nil {
		return 0
	}

	return int16(num)
}

func FormatFloat64(number float64) string {
	return strconv.FormatFloat(number, 'f', -1, 64)
}

func FormatFloat2(number float64) string {
	return strconv.FormatFloat(number, 'f', 2, 64)
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func PercentageOf(percentage float64, total float64) float64 {
	if percentage == 0 || total == 0 {
		return 0
	}

	if percentage == 100 {
		return total
	}

	return percentage * total / 100
}

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
