package lib

import ()

// TODO: field validation
var regexPatterns = map[string]string{
	"obscenities": ``,
}

func ValidateField(field, content string) bool { return true }
