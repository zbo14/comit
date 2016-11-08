package lib

import (
	re "regexp"
)

// TODO: field validation

var regexPatterns = map[string]string{
	"obscenities": ``,
}

func ValidateField(content string) bool {
	pattern := re.MustCompile(regexPatterns["obscenities"])
	obscene := pattern.MatchString(content)
	if obscene {
		return false
	}
	return true
}
