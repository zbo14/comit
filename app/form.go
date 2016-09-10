package app

import (
	"crypto/md5"
	"errors"
	"fmt"
	lib "github.com/zballs/3ii/lib"
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
		(*form).Pubkey = lib.SERVICE.ReadPubkeyString(str)
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

//=========================================//

func Resolve(tm time.Time) Item {
	return func(form *Form) error {
		(*form).Resolved = tm
		(*form).ResponseTime = tm.Sub((*form).Time).Hours()
		return nil
	}
}
