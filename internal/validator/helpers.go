package validator

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

func MinRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

func MaxRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

var RgxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
