package lib

import (
	"fmt"
	util "github.com/zballs/3ii/util"
	re "regexp"
	"strings"
)

type Detail struct{}

type ServiceDetail struct {
	detail  string
	options []string
}

type DetailInterface interface {
	Read(str string, sd *ServiceDetail) string
	Write(str string, sd *ServiceDetail) string
}

func (Detail) Read(str string, sd *ServiceDetail) string {
	detail := util.RegexQuestionMarks(sd.Detail())
	options := strings.Join(sd.Options(), `|`)
	res := re.MustCompile(fmt.Sprintf(`%v {(%v)}`, detail, options)).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func (Detail) Write(str string, sd *ServiceDetail) string {
	detail := sd.Detail()
	return fmt.Sprintf("%v {%v}", detail, str)
}

var DETAIL DetailInterface = Detail{}

func (sd *ServiceDetail) Detail() string {
	return sd.detail
}

func (sd *ServiceDetail) Options() []string {
	return sd.options
}

// Service Details

var completelyOut = &ServiceDetail{
	detail:  "completely out?",
	options: []string{"yes", "no"},
}

var potholeLocation = &ServiceDetail{
	detail:  "pothole location",
	options: []string{"bike lane", "crosswalk", "curb lane", "intersection", "traffic lane"},
}

var backyardBaited = &ServiceDetail{
	detail:  "backyard baited?",
	options: []string{"yes", "no"},
}
