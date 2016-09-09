package lib

import (
	"fmt"
	util "github.com/zballs/3ii/util"
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	ReadType(tx []byte) []byte
	WriteType(tx []byte) []byte
	ReadAddress(tx []byte) []byte
	WriteAddress(tx []byte) []byte
	ReadDescription(tx []byte) []byte
	WriteDescription(tx []byte) []byte
	ReadSpecField(tx []byte, t []byte) []byte
	WriteSpecField(tx []byte, t []byte) []byte
	ReadPubkeyBytes(tx []byte) []byte
	WritePrivkeyBytes(tx []byte) []byte
}

var SpecFields = map[string]FieldGroup{
	"street light out":             CompletelyOut,
	"pothole in street":            PotholeLocation,
	"rodent baiting/rat complaint": BackyardBaited,
	"tree trim":                    nil,
}

func (Service) ReadType(tx []byte) []byte {
	return re.MustCompile(`type{([\w+\s]+)}`).FindSubmatch(tx)[1]
}

func (Service) WriteType(tx []byte) []byte {
	return []byte(fmt.Sprintf("type{%v}", string(tx)))
}

func (Service) ReadAddress(tx []byte) []byte {
	return re.MustCompile(`address{([\w\s'\-\.\,]+)}`).FindSubmatch(tx)[1]
}

func (Service) WriteAddress(tx []byte) []byte {
	return []byte(fmt.Sprintf("address{%v}", string(tx)))
}

func (Service) ReadDescription(tx []byte) []byte {
	return re.MustCompile(`description{([\w+'?\w?.?\s]+)}`).FindSubmatch(tx)[1]
}

func (Service) WriteDescription(tx []byte) []byte {
	return []byte(fmt.Sprintf("description{%v}", string(tx)))
}

func (Service) ReadSpecField(tx []byte, t []byte) []byte {
	return SpecFields[string(t)]["read"](tx)
}

func (Service) WriteSpecField(tx []byte, t []byte) []byte {
	return SpecFields[string(t)]["write"](tx)
}

func (Service) ReadPubkeyBytes(tx []byte) []byte {
	return util.ReadPubKeyBytes(tx)
}

func (Service) WritePrivkeyBytes(tx []byte) []byte {
	return util.WritePrivKeyBytes(tx)
}

var SERVICE ServiceInterface = Service{}
