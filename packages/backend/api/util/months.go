package util

import (
	"strings"
)

func GetMonthIfContains(str string) int16 {

	str = strings.ToLower(str)

	if strings.Contains(str, "enero") || strings.Contains(str, "ene") {
		return 1
	}
	if strings.Contains(str, "febrero") || strings.Contains(str, "feb") {
		return 2
	}
	if strings.Contains(str, "marzo") || strings.Contains(str, "mar") {
		return 3
	}
	if strings.Contains(str, "abril") || strings.Contains(str, "abr") {
		return 4
	}
	if strings.Contains(str, "mayo") || strings.Contains(str, "may") {
		return 5
	}
	if strings.Contains(str, "junio") || strings.Contains(str, "jun") {
		return 6
	}
	if strings.Contains(str, "julio") || strings.Contains(str, "jul") {
		return 7
	}
	if strings.Contains(str, "agosto") || strings.Contains(str, "ago") {
		return 8
	}
	if strings.Contains(str, "septiembre") || strings.Contains(str, "sep") {
		return 9
	}
	if strings.Contains(str, "octubre") || strings.Contains(str, "oct") {
		return 10
	}
	if strings.Contains(str, "noviembre") || strings.Contains(str, "nov") {
		return 11
	}
	if strings.Contains(str, "diciembre") || strings.Contains(str, "dic") {
		return 12
	}

	return 0

}

func FromInt16ToMonth(month int16) string {

	switch month {
	case 1:
		return "Enero"
	case 2:
		return "Febrero"
	case 3:
		return "Marzo"
	case 4:
		return "Abril"
	case 5:
		return "Mayo"
	case 6:
		return "Junio"
	case 7:
		return "Julio"
	case 8:
		return "Agosto"
	case 9:
		return "Septiembre"
	case 10:
		return "Octubre"
	case 11:
		return "Noviembre"
	case 12:
		return "Diciembre"
	}

	return ""
}

func MonthToInt16(month string) int16 {
	month = strings.ToLower(month)
	switch month {
	case "enero", "ene":
		return 1
	case "febrero", "feb":
		return 2
	case "marzo", "mar":
		return 3
	case "abril", "abr":
		return 4
	case "mayo", "may":
		return 5
	case "junio", "jun":
		return 6
	case "julio", "jul":
		return 7
	case "agosto", "ago":
		return 8
	case "septiembre", "sep":
		return 9
	case "octubre", "oct":
		return 10
	case "noviembre", "nov":
		return 11
	case "diciembre", "dic":
		return 12
	}

	return 0
}
