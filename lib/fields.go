package lib

import (
	"fmt"
	util "github.com/zballs/3ii/util"
	re "regexp"
	"strings"
)

type Field struct{}

type FieldOptions struct {
	field   string
	options []string
}

type FieldInterface interface {
	ReadField(str string, fieldOpts *FieldOptions) string
	WriteField(str string, fieldOpts *FieldOptions) string
}

func (Field) ReadField(str string, fieldOpts *FieldOptions) string {
	field := util.RegexQuestionMarks(fieldOpts.GetField())
	options := strings.Join(fieldOpts.GetOptions(), `|`)
	res := re.MustCompile(fmt.Sprintf(`%v {(%v)}`, field, options)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Field) WriteField(str string, fieldOpts *FieldOptions) string {
	fmt.Println(str)
	field := fieldOpts.GetField()
	return fmt.Sprintf("%v {%v}", field, str)
}

var FIELD FieldInterface = Field{}

func (fieldOpt FieldOptions) GetField() string {
	return fieldOpt.field
}

func (fieldOpt FieldOptions) GetOptions() []string {
	return fieldOpt.options
}

// Field Options

var completelyOut = &FieldOptions{
	field:   "completely out?",
	options: []string{"yes", "no"},
}

var potholeLocation = &FieldOptions{
	field:   "pothole location",
	options: []string{"bike lane", "crosswalk", "curb lane", "intersection", "traffic lane"},
}

var backyardBaited = &FieldOptions{
	field:   "backyard baited?",
	options: []string{"yes", "no"},
}
