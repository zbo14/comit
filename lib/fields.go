package lib

import (
	"fmt"
	util "github.com/zballs/3ii/util"
	re "regexp"
	"strings"
)

type Field struct{}

type FieldOptions struct {
	Field   string
	Options []string
}

type FieldInterface interface {
	ReadField(str string, fieldOpts *FieldOptions) string
	WriteField(str string, fieldOpts *FieldOptions) string
}

func (Field) ReadField(str string, fieldOpts *FieldOptions) string {
	field := util.RegexQuestionMarks(fieldOpts.Field)
	options := strings.Join(fieldOpts.Options, `|`)
	res := re.MustCompile(fmt.Sprintf(`%v {(%v)}`, field, options)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Field) WriteField(str string, fieldOpts *FieldOptions) string {
	field := fieldOpts.Field
	return fmt.Sprintf("%v {%v}", field, str)
}

var FIELD FieldInterface = Field{}

// Field Options

var CompletelyOut = &FieldOptions{
	Field:   "completely out?",
	Options: []string{"yes", "no"},
}

var PotholeLocation = &FieldOptions{
	Field:   "pothole location",
	Options: []string{"bike lane", "crosswalk", "curb lane", "intersection", "traffic lane"},
}

var BackyardBaited = &FieldOptions{
	Field:   "backyard baited?",
	Options: []string{"yes", "no"},
}
