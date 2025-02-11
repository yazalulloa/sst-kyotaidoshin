package util

import (
	"fmt"
	"strings"
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
