package lib

import (
	"fmt"
	util "github.com/zballs/3ii/util"
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	ReadType(tx string) string
	WriteType(str string) string
	ReadAddress(str string) string
	WriteAddress(str string) string
	ReadDescription(str string) string
	WriteDescription(str string) string
	ReadSpecField(str string, _type string) string
	WriteSpecField(str string, _type string) string
	WritePubkeyString(str string) string
	WritePrivkeyString(str string) string
	ReadPubkeyString(str string) string
}

var SpecFields = map[string]FieldGroup{
	"street light out":             CompletelyOut,
	"pothole in street":            PotholeLocation,
	"rodent baiting/rat complaint": BackyardBaited,
	"tree trim":                    nil,
}

func (Service) ReadType(str string) string {
	return re.MustCompile(`type{([\w+\s]+)}`).FindStringSubmatch(str)[1]
}

func (Service) WriteType(str string) string {
	return fmt.Sprintf("type{%v}", str)
}

func (Service) ReadAddress(str string) string {
	return re.MustCompile(`address{([\w\s'\-\.\,]+)}`).FindStringSubmatch(str)[1]
}

func (Service) WriteAddress(str string) string {
	return fmt.Sprintf("address{%v}", str)
}

func (Service) ReadDescription(str string) string {
	return re.MustCompile(`description{([\w+'?\w?.?\s]+)}`).FindStringSubmatch(str)[1]
}

func (Service) WriteDescription(str string) string {
	return fmt.Sprintf("description{%v}", str)
}

func (Service) ReadSpecField(str string, _type string) string {
	return SpecFields[_type]["read"](str)
}

func (Service) WriteSpecField(str string, _type string) string {
	return SpecFields[_type]["write"](str)
}

func (Service) WritePubkeyString(str string) string {
	return util.WritePubKeyString(str)
}

func (Service) ReadPubkeyString(str string) string {
	return util.ReadPubKeyString(str)
}

func (Service) WritePrivkeyString(str string) string {
	return util.WritePrivKeyString(str)
}

var SERVICE ServiceInterface = Service{}
