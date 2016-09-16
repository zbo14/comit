package lib

import (
	"bytes"
	"fmt"
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	Regex(field string) string
	FieldOpts(_type string) *FieldOptions
	ReadField(str string, field string) string
	WriteField(str string, field string) string
	ReadSpecField(str string, _type string) string
	WriteSpecField(str string, _type string) string
	FormatFieldOpts(_type string) (string, string)
}

var ServiceOptions = map[string]*FieldOptions{
	"street light out":             CompletelyOut,
	"pothole in street":            PotholeLocation,
	"rodent baiting/rat complaint": BackyardBaited,
	"tree trim":                    nil,
	"garbage cart black maintenance/replacement": nil,
}

var ServiceDepts = map[string]string{
	"street light out":             "infrastructure",
	"pothole in street":            "infrastructure",
	"rodent baiting/rat complaint": "i dont know",
	"tree trim":                    "i dont know",
	"garbage cart black maintenance/replacement": "sanitation",
}

var RegexPatterns = map[string]string{
	"type":        `[\w\s]+`,
	"address":     `[\w\s'\-\.\,]+`,
	"description": `[\w\s'\-\.\,\?\!\\]+`,
}

func (Service) FieldOpts(_type string) *FieldOptions {
	return ServiceOptions[_type]
}

func (Service) Regex(field string) string {
	return RegexPatterns[field]
}

func (serv Service) ReadField(str string, field string) string {
	pattern := serv.Regex(field)
	res := re.MustCompile(fmt.Sprintf(`%v{(%v)}`, field, pattern)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Service) WriteField(str string, field string) string {
	return fmt.Sprintf(`%v{%v}`, field, str)
}

func (serv Service) ReadSpecField(str string, _type string) string {
	fieldOpts := serv.FieldOpts(_type)
	if fieldOpts == nil {
		return ""
	}
	return FIELD.ReadField(str, fieldOpts)
}

func (serv Service) WriteSpecField(str string, _type string) string {
	fieldOpts := serv.FieldOpts(_type)
	if fieldOpts == nil {
		return ""
	}
	return FIELD.WriteField(str, fieldOpts)
}

func (serv Service) FormatFieldOpts(_type string) (string, string) {
	fieldOpts := serv.FieldOpts(_type)
	if fieldOpts == nil {
		return "No Options", ""
	}
	var Bytes bytes.Buffer
	for _, opt := range fieldOpts.Options {
		Bytes.WriteString(fmt.Sprintf(`<option value="%v">%v</option>`, opt, opt))
	}
	return fieldOpts.Field, Bytes.String()
}

var SERVICE ServiceInterface = Service{}
