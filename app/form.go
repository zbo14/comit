package app

import (
	"crypto/md5"
	"errors"
	"fmt"
	lib "github.com/zballs/3ii/lib"
	util "github.com/zballs/3ii/util"
	"time"
)

type Form struct {
	Time        time.Time
	Type        string
	Address     string
	Description string
	SpecField   string
	Pubkey      string

	//==============//

	Resolved     time.Time
	ResponseTime float64
}

type Formlist []*Form

type Item func(*Form) error

func NewForm(items ...Item) (*Form, error) {
	form := &Form{}
	for _, item := range items {
		err := item(form)
		if err != nil {
			return nil, err
		}
	}
	return form, nil
}

func Time(tm time.Time) Item {
	return func(form *Form) error {
		(*form).Time = tm
		return nil
	}
}

func Type(str string) Item {
	return func(form *Form) error {
		(*form).Type = lib.SERVICE.ReadType(str)
		return nil
	}
}

func Address(str string) Item {
	return func(form *Form) error {
		(*form).Address = lib.SERVICE.ReadAddress(str)
		return nil
	}
}

func Description(str string) Item {
	return func(form *Form) error {
		(*form).Description = lib.SERVICE.ReadDescription(str)
		return nil
	}
}

func SpecField(str string) Item {
	return func(form *Form) error {
		_type := (*form).Type
		if len(_type) > 0 {
			(*form).SpecField = lib.SERVICE.ReadSpecField(str, _type)
			return nil
		}
		return errors.New("cannot set form details without type")
	}
}

func Pubkey(str string) Item {
	return func(form *Form) error {
		(*form).Pubkey = util.ReadPubKeyString(str)
		return nil
	}
}

func MakeForm(str string) (*Form, error) {
	form, err := NewForm(
		Time(time.Now()),
		Type(str),
		Address(str),
		Description(str),
		SpecField(str),
		Pubkey(str),
	)
	if err != nil {
		return nil, err
	}
	return form, nil
}

func CheckStatus(tm time.Time) string {
	var nilTime = time.Time{}
	if tm == nilTime {
		return "unresolved"
	}
	return "resolved " + tm.String()[:16]
}

func ParseForm(form *Form) string {
	_posted := (*form).Time.String()[:16] // up to the minute
	_type := lib.SERVICE.WriteType((*form).Type)
	_address := lib.SERVICE.WriteAddress((*form).Address)
	_description := lib.SERVICE.WriteDescription((*form).Description)
	_specfield := lib.SERVICE.WriteSpecField((*form).SpecField, (*form).Type)
	_pubkey := util.WritePubKeyString((*form).Pubkey)
	_resolved := CheckStatus((*form).Resolved)
	return _posted + "<br>" + _type + "<br>" + _address + "<br>" + _description + "<br>" + _specfield + "<br>" + _pubkey + "<br>" + _resolved + "<br><br>"
}

func FormID(form *Form) string {
	bytes := make([]byte, 32)
	items := []string{
		(*form).Type,
		(*form).Address,
		(*form).Description,
		(*form).SpecField,
		(*form).Pubkey,
	}
	for _, item := range items {
		for idx, _ := range bytes {
			if idx < len(item) {
				bytes[idx] ^= byte(item[idx])
			} else {
				break
			}
		}
	}
	return fmt.Sprintf("%x", md5.Sum(bytes))
}

func MatchForm(str string, form *Form) bool {
	_type := lib.SERVICE.ReadType(str)
	if len(_type) > 0 {
		if !(_type == (*form).Type) {
			return false
		}
	}
	_address := lib.SERVICE.ReadAddress(str)
	if len(_address) > 0 {
		if !util.SubstringMatch(_address, (*form).Address) {
			return false
		}
	}
	_specfield := lib.SERVICE.ReadSpecField(str, _type)
	if len(_specfield) > 0 {
		if !(_specfield == (*form).SpecField) {
			return false
		}
	}
	return true
}

//=========================================//

func Resolve(tm time.Time) Item {
	return func(form *Form) error {
		(*form).Resolved = tm
		(*form).ResponseTime = tm.Sub((*form).Time).Hours()
		return nil
	}
}
