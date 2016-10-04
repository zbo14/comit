package util

import (
	re "regexp"
	"strings"
)

// Substring Match

func SubstringMatch(substr string, str string) bool {
	match := re.MustCompile(strings.ToLower(substr)).FindString(strings.ToLower(str))
	if len(match) > 0 {
		return true
	}
	return false
}

// Regex Formatting

func RegexQuestionMarks(str string) string {
	return `` + strings.Replace(str, `?`, `\?`, -1)
}

// HTML

func ExtractText(str string) string {
	return re.MustCompile(`>(.*?)<`).FindStringSubmatch(str)[1]
}
