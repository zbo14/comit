package lib

import (
	. "github.com/tendermint/go-common"
	re "regexp"
)

var regexPatterns = map[string]string{
	"issue":       `[\w\s\/]+`,
	"location":    `[\w\s'\-\.\,]+`,
	"description": `[\w\s'\-\.\,\?\!\/]+`,
	"before":      `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
	"after":       `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
	"status":      `resolved|unresolved`,
}

func ReadField(field, content string) string {
	pattern := regexPatterns(field)
	res := re.MustCompile(Fmt(`%v {(%v)}`, field, pattern)).FindStringSubmatch(content)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func WriteField(field, content string) string {
	return Fmt("%v {%v}", field, str)
}
