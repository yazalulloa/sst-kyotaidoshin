package util

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"log"
	"reflect"
	"strings"
	"sync"
)

const OneKb = 1024.0
const OneMb = 1024 * OneKb
const OneGb = 1024 * OneMb

type SortOrderType string

const (
	SortOrderTypeASC  SortOrderType = "ASC"
	SortOrderTypeDESC SortOrderType = "DESC"
)

type AllowedCurrencies string

const (
	AllowedCurrenciesVED AllowedCurrencies = "VED"
	AllowedCurrenciesUSD AllowedCurrencies = "USD"
)

func AllowedCurrenciesStringArray() []string {
	return []string{
		"VED",
		"USD",
	}
}

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

		log.Printf("Validator validatorInstance created: %v", validatorInstance != nil)
	})

	log.Printf("Returning validator validatorInstance: %v", validatorInstance != nil)

	return validatorInstance, err
}
