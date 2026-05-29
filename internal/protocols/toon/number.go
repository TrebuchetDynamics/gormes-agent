package toon

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$`)
var numericLikePattern = regexp.MustCompile(`^-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?$`)
var forbiddenLeadingZeroPattern = regexp.MustCompile(`^-?0\d+`)

func canonicalNumber(input string) (string, error) {
	s := strings.TrimSpace(input)
	if !jsonNumberPattern.MatchString(s) {
		return "", fmt.Errorf("toon: invalid JSON number %q", input)
	}
	sign := false
	if strings.HasPrefix(s, "-") {
		sign = true
		s = s[1:]
	}

	exp := 0
	if index := strings.IndexAny(s, "eE"); index >= 0 {
		rawExp := s[index+1:]
		parsed, err := strconv.Atoi(rawExp)
		if err != nil {
			return "", fmt.Errorf("toon: invalid number exponent %q", input)
		}
		exp = parsed
		s = s[:index]
	}

	intPart := s
	fracPart := ""
	if index := strings.IndexByte(s, '.'); index >= 0 {
		intPart = s[:index]
		fracPart = s[index+1:]
	}
	digits := intPart + fracPart
	if strings.Trim(digits, "0") == "" {
		return "0", nil
	}

	decimalAt := len(intPart) + exp
	var out string
	switch {
	case decimalAt <= 0:
		out = "0." + strings.Repeat("0", -decimalAt) + digits
	case decimalAt >= len(digits):
		out = digits + strings.Repeat("0", decimalAt-len(digits))
	default:
		out = digits[:decimalAt] + "." + digits[decimalAt:]
	}

	if dot := strings.IndexByte(out, '.'); dot >= 0 {
		left := strings.TrimLeft(out[:dot], "0")
		if left == "" {
			left = "0"
		}
		right := strings.TrimRight(out[dot+1:], "0")
		if right == "" {
			out = left
		} else {
			out = left + "." + right
		}
	} else {
		out = strings.TrimLeft(out, "0")
		if out == "" {
			out = "0"
		}
	}

	if sign && out != "0" {
		out = "-" + out
	}
	return out, nil
}

func isNumericToken(s string) bool {
	return jsonNumberPattern.MatchString(s) && !forbiddenLeadingZeroPattern.MatchString(s)
}
