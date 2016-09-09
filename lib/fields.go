package lib

import (
	"fmt"
	re "regexp"
)

type Field struct{}

type FieldMethod func(tx []byte) []byte

type FieldInterface interface {
	ReadCompletelyOut(tx []byte) []byte
	WriteCompletelyOut(tx []byte) []byte
	ReadPotholeLocation(tx []byte) []byte
	WritePotholeLocation(tx []byte) []byte
	ReadBackyardBaited(tx []byte) []byte
	WriteBackyardBaited(tx []byte) []byte
}

type FieldGroup map[string]FieldMethod

func (Field) ReadCompletelyOut(tx []byte) []byte {
	return re.MustCompile(`completely-out{(yes|no)}`).FindSubmatch(tx)[1]
}

func (Field) WriteCompletelyOut(tx []byte) []byte {
	return []byte(fmt.Sprintf("completely-out{%v}", string(tx)))
}

func (Field) ReadPotholeLocation(tx []byte) []byte {
	return re.MustCompile(`pothole-location{(bike\slane|crosswalk|curb\slane|intersection|traffic\slane)}`).FindSubmatch(tx)[1]
}

func (Field) WritePotholeLocation(tx []byte) []byte {
	return []byte(fmt.Sprintf("pothole-location{%v}", string(tx)))
}

func (Field) ReadBackyardBaited(tx []byte) []byte {
	return re.MustCompile(`backyard-baited{(yes|no)}`).FindSubmatch(tx)[1]
}

func (Field) WriteBackyardBaited(tx []byte) []byte {
	return []byte(fmt.Sprintf("backyard-baited{%v}", string(tx)))
}

var FIELD FieldInterface = Field{}

var CompletelyOut = FieldGroup{
	"read":  FIELD.ReadCompletelyOut,
	"write": FIELD.WriteCompletelyOut,
}

var PotholeLocation = FieldGroup{
	"read":  FIELD.ReadPotholeLocation,
	"write": FIELD.WritePotholeLocation,
}

var BackyardBaited = FieldGroup{
	"read":  FIELD.ReadBackyardBaited,
	"write": FIELD.WriteBackyardBaited,
}
