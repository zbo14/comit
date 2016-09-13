package lib

import (
	"fmt"
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
	field := fieldOpts.Field
	options := strings.Join(fieldOpts.Options, `|`)
	res := re.MustCompile(fmt.Sprintf(`%v{(%v)}`, field, options)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Field) WriteField(str string, fieldOpts *FieldOptions) string {
	field := fieldOpts.Field
	return fmt.Sprintf("%v{%v}", field, str)
}

var FIELD FieldInterface = Field{}

// Field Options

var CompletelyOut = &FieldOptions{
	Field:   "Completely Out?",
	Options: []string{"yes", "no"},
}

var PotholeLocation = &FieldOptions{
	Field:   "Pothole Location",
	Options: []string{"bike lane", "crosswalk", "curb lane", "intersection", "traffic lane"},
}

var BackyardBaited = &FieldOptions{
	Field:   "Backyard Baited?",
	Options: []string{"yes", "no"},
}
