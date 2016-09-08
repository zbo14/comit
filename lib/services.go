package lib

import (
	re "regexp"
)

type Service struct{}

type ServiceInterface interface {
	Type(tx []byte) []byte
	Address(tx []byte) []byte
	Description(tx []byte) []byte
	SpecField(tx []byte) []byte
	Pubkey(tx []byte) []byte
}

var SpecFields = map[string]FieldMethod{
	"txeet light out":              FIELD.CompletelyOut,
	"pothole in txeet":             FIELD.PotholeLocation,
	"rodent baiting/rat complaint": FIELD.BackyardBaited,
	"tree trim":                    nil,
}

func (Service) Type(tx []byte) []byte {
	return re.MustCompile(`type:([\w+\s]+)\n`).FindSubmatch(tx)[1]
}

func (Service) Address(tx []byte) []byte {
	return re.MustCompile(`address:([\w\s'\-\.\,]+)\n`).FindSubmatch(tx)[1]
}

func (Service) Description(tx []byte) []byte {
	return re.MustCompile(`description:([\w+'?\w?.?\s]+)\n`).FindSubmatch(tx)[1]
}

func (serv Service) SpecField(tx []byte) []byte {
	return SpecFields[string(serv.Type(tx))](tx)
}

func (Service) Pubkey(tx []byte) []byte {
	return re.MustCompile(`pubkey:([\w]+)$`).FindSubmatch(tx)[1]
}

var SERVICE ServiceInterface = Service{}
