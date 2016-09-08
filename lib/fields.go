package lib

import (
	re "regexp"
)

type Field struct{}

type FieldMethod func(tx []byte) []byte

type FieldInterface interface {
	CompletelyOut(tx []byte) []byte
	PotholeLocation(tx []byte) []byte
	BackyardBaited(tx []byte) []byte
}

func (Field) CompletelyOut(tx []byte) []byte {
	return re.MustCompile(`completely-out:(yes|no)\n`).FindSubmatch(tx)[1]
}

func (Field) PotholeLocation(tx []byte) []byte {
	return re.MustCompile(`pothole-location:(bike\slane|crosswalk|curb\slane|intersection|traffic\slane)\n`).FindSubmatch(tx)[1]
}

func (Field) BackyardBaited(tx []byte) []byte {
	return re.MustCompile(`backyard-baited:(yes|no)\n`).FindSubmatch(tx)[1]
}

var FIELD FieldInterface = Field{}
