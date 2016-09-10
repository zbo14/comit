package lib

import (
	"fmt"
	re "regexp"
)

type Field struct{}

type FieldMethod func(str string) string

type FieldInterface interface {
	ReadCompletelyOut(str string) string
	WriteCompletelyOut(str string) string
	ReadPotholeLocation(str string) string
	WritePotholeLocation(str string) string
	ReadBackyardBaited(str string) string
	WriteBackyardBaited(str string) string
}

type FieldGroup map[string]FieldMethod

func (Field) ReadCompletelyOut(str string) string {
	return re.MustCompile(`completely-out{(yes|no)}`).FindStringSubmatch(str)[1]
}

func (Field) WriteCompletelyOut(str string) string {
	return fmt.Sprintf("completely-out{%v}", str)
}

func (Field) ReadPotholeLocation(str string) string {
	return re.MustCompile(`pothole-location{(bike\slane|crosswalk|curb\slane|intersection|traffic\slane)}`).FindStringSubmatch(str)[1]
}

func (Field) WritePotholeLocation(str string) string {
	return fmt.Sprintf("pothole-location{%v}", str)
}

func (Field) ReadBackyardBaited(str string) string {
	return re.MustCompile(`backyard-baited{(yes|no)}`).FindStringSubmatch(str)[1]
}

func (Field) WriteBackyardBaited(str string) string {
	return fmt.Sprintf("backyard-baited{%v}", str)
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
