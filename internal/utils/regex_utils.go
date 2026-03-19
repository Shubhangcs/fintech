package utils

import (
	"regexp"
	"unicode"
)

var (
	EmailRegx   = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	PhoneRegx   = regexp.MustCompile(`^[6-9]\d{9}$`)
	AadharRegx  = regexp.MustCompile(`^\d{12}$`)
	PanRegx     = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)
	PincodeRegx = regexp.MustCompile(`^\d{6}$`)
)

func IsValid(re *regexp.Regexp, data string) bool {
	return re.MatchString(data)
}

// IsValidPassword checks: min 8 chars, at least one lowercase, uppercase, digit, and special char.
func IsValidPassword(p string) bool {
	if len(p) < 8 {
		return false
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	special := "@$!%*?&"
	for _, c := range p {
		switch {
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasDigit = true
		case containsRune(special, c):
			hasSpecial = true
		}
	}
	return hasLower && hasUpper && hasDigit && hasSpecial
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
