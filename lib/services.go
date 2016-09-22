package lib

import (
	"bytes"
	"fmt"
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	GetServices() []string
	GetDepts() map[string]*struct{}
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

var serviceOptions = map[string]*FieldOptions{
	"street light out":             completelyOut,
	"pothole in street":            potholeLocation,
	"rodent baiting/rat complaint": backyardBaited,
	"tree trim":                    nil,
	"garbage cart black maintenance/replacement": nil,
}

var serviceDepts = map[string]string{
	"street light out":             "infrastructure",
	"pothole in street":            "infrastructure",
	"rodent baiting/rat complaint": "i dont know",
	"tree trim":                    "i dont know",
	"garbage cart black maintenance/replacement": "sanitation",
}

var regexPatterns = map[string]string{
	"service":     `[\w\s]+`,
	"address":     `[\w\s'\-\.\,]+`,
	"description": `[\w\s'\-\.\,\?\!\\]+`,
	"before":      `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
	"after":       `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
}

func (Service) GetServices() []string {
	services := make([]string, len(serviceDepts))
	idx := 0
	for service, _ := range serviceDepts {
		services[idx] = service
		idx++
	}
	return services
}

func (Service) GetDepts() map[string]*struct{} {
	depts := make(map[string]*struct{})
	for _, dept := range serviceDepts {
		depts[dept] = nil
	}
	return depts
}

func (Service) FieldOpts(service string) *FieldOptions {
	return serviceOptions[service]
}

func (Service) ServiceDept(service string) string {
	return serviceDepts[service]
}

func (Service) DeptServices(dept string) (services []string) {
	for s, d := range serviceDepts {
		if dept == d {
			services = append(services, s)
		}
	}
	return
}

func (Service) Regex(field string) string {
	return regexPatterns[field]
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
	for _, opt := range fieldOpts.GetOptions() {
		Bytes.WriteString(fmt.Sprintf(`<option value="%v">%v</option>`, opt, opt))
	}
	return fieldOpts.GetField(), Bytes.String()
}

var SERVICE ServiceInterface = Service{}
