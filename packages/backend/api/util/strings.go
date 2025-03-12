package util

import "regexp"

var nonAlphanumericRegex = regexp.MustCompile(`[^\p{L}\p{N} ]+`)
var nonNumericRegex = regexp.MustCompile(`[^\p{N} ]+`)

func RemoveNonAlphanumeric(s string) string {
	return nonAlphanumericRegex.ReplaceAllString(s, "")
}

func RemoveNonNumeric(s string) string {
	return nonNumericRegex.ReplaceAllString(s, "")
}
