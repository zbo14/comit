package lib

import (
	"bytes"
	"fmt"
	re "regexp"
)

// TODO: change to complaint/issue/incident

type Service struct{}

type ServiceInterface interface {
	Services() []string
	Depts() map[string]*struct{}
	Regex(field string) string
	ServiceDetail(service string) *ServiceDetail
	ServiceDept(service string) string
	DeptServices(dept string) []string
	ReadField(str string, field string) string
	WriteField(str string, field string) string
	ReadDetail(str string, service string) string
	WriteDetail(str string, service string) string
	FormatDetail(service string) (string, string)
}

var serviceDetails = map[string]*ServiceDetail{
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

/* Map or list?
var issueMap = map[string]string{
	"police officer complaint": "public safety",
}

var issueList = []string{
	"police complaint",
}
*/

var regexPatterns = map[string]string{
	"service":     `[\w\s\/]+`,
	"address":     `[\w\s'\-\.\,]+`,
	"description": `[\w\s'\-\.\,\?\!\/]+`,
	"before":      `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
	"after":       `\d{4}-\d{2}-\d{2}T\w{2}\:\d{2}:\d{2}`,
	"status":      `resolved|unresolved`,
}

func (Service) Services() []string {
	services := make([]string, len(serviceDepts))
	idx := 0
	for service, _ := range serviceDepts {
		services[idx] = service
		idx++
	}
	return services
}

func (Service) Depts() map[string]*struct{} {
	depts := make(map[string]*struct{})
	for _, dept := range serviceDepts {
		depts[dept] = nil
	}
	return depts
}

func (Service) ServiceDetail(service string) *ServiceDetail {
	return serviceDetails[service]
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

func (s Service) ReadField(str string, field string) string {
	pattern := s.Regex(field)
	res := re.MustCompile(fmt.Sprintf(`%v {(%v)}`, field, pattern)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Service) WriteField(str string, field string) string {
	return fmt.Sprintf("%v {%v}", field, str)
}

func (s Service) ReadDetail(str string, service string) string {
	sd := s.ServiceDetail(service)
	if sd == nil {
		return ""
	}
	return DETAIL.Read(str, sd)
}

func (s Service) WriteDetail(str string, service string) string {
	sd := s.ServiceDetail(service)
	if sd == nil {
		return ""
	}
	return DETAIL.Write(str, sd)
}

func (s Service) FormatDetail(service string) (string, string) {
	sd := s.ServiceDetail(service)
	if sd == nil {
		return "no options", ""
	}
	var Bytes bytes.Buffer
	Bytes.WriteString(`<option value="">--</option>`)
	for _, opt := range sd.Options {
		Bytes.WriteString(fmt.Sprintf(`<option value="%v">%v</option>`, opt, opt))
	}
	return sd.Detail, Bytes.String()
}

var SERVICE ServiceInterface = Service{}
