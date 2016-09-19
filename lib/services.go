package lib

import (
	"bytes"
	"fmt"
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	Regex(field string) string
	FieldOpts(service string) *FieldOptions
	ServiceDept(service string) string
	DeptServices(dept string) []string
	ReadField(str string, field string) string
	WriteField(str string, field string) string
	ReadSpecField(str string, service string) string
	WriteSpecField(str string, service string) string
	FormatFieldOpts(service string) (string, string)
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
	"service":     `[\w\s]+`,
	"address":     `[\w\s'\-\.\,]+`,
	"description": `[\w\s'\-\.\,\?\!\\]+`,
}

func (Service) FieldOpts(service string) *FieldOptions {
	return ServiceOptions[service]
}

func (Service) ServiceDept(service string) string {
	return ServiceDepts[service]
}

func (Service) DeptServices(dept string) (services []string) {
	for service, _dept := range ServiceDepts {
		if dept == _dept {
			services = append(services, service)
		}
	}
	return
}

func (Service) Regex(field string) string {
	return RegexPatterns[field]
}

func (serv Service) ReadField(str string, field string) string {
	pattern := serv.Regex(field)
	res := re.MustCompile(fmt.Sprintf(`%v {(%v)}`, field, pattern)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Service) WriteField(str string, field string) string {
	return fmt.Sprintf("%v {%v}", field, str)
}

func (serv Service) ReadSpecField(str string, service string) string {
	fieldOpts := serv.FieldOpts(service)
	if fieldOpts == nil {
		return ""
	}
	return FIELD.ReadField(str, fieldOpts)
}

func (serv Service) WriteSpecField(str string, service string) string {
	fieldOpts := serv.FieldOpts(service)
	if fieldOpts == nil {
		return ""
	}
	return FIELD.WriteField(str, fieldOpts)
}

func (serv Service) FormatFieldOpts(service string) (string, string) {
	fieldOpts := serv.FieldOpts(service)
	if fieldOpts == nil {
		return "no options", ""
	}
	var Bytes bytes.Buffer
	Bytes.WriteString(`<option value="">--</option>`)
	for _, opt := range fieldOpts.Options {
		Bytes.WriteString(fmt.Sprintf(`<option value="%v">%v</option>`, opt, opt))
	}
	return fieldOpts.Field, Bytes.String()
}

var SERVICE ServiceInterface = Service{}
