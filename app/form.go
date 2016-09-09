package app

import (
	"crypto/md5"
	"fmt"
	lib "github.com/zballs/3ii/lib"
	"time"
)

type Form struct {
	Time        time.Time
	Type        []byte
	Address     []byte
	Description []byte
	SpecField   []byte
	PubkeyBytes []byte

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

func Type(tx []byte) Item {
	return func(form *Form) error {
		(*form).Type = lib.SERVICE.Type(tx)
		return nil
	}
}

func Address(tx []byte) Item {
	return func(form *Form) error {
		(*form).Address = lib.SERVICE.Address(tx)
		return nil
	}
}

func Description(tx []byte) Item {
	return func(form *Form) error {
		(*form).Description = lib.SERVICE.Description(tx)
		return nil
	}
}

func SpecField(tx []byte) Item {
	return func(form *Form) error {
		(*form).SpecField = lib.SERVICE.SpecField(tx)
		return nil
	}
}

func PubkeyBytes(tx []byte) Item {
	return func(form *Form) error {
		(*form).PubkeyBytes = lib.SERVICE.PubkeyBytes(tx)
		return nil
	}
}

func MakeForm(tx []byte) (*Form, error) {
	form, err := NewForm(
		Time(time.Now()),
		Type(tx),
		Address(tx),
		Description(tx),
		SpecField(tx),
		PubkeyBytes(tx),
	)
	if err != nil {
		return nil, err
	}
	return form, nil
}

func FormID(form *Form) string {
	bytes := make([]byte, 32)
	items := [][]byte{
		(*form).Type,
		(*form).Address,
		(*form).Description,
		(*form).SpecField,
		(*form).PubkeyBytes,
	}
	for _, item := range items {
		for idx, _ := range bytes {
			if idx < len(item) {
				bytes[idx] ^= item[idx]
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
