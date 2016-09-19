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
	Service     string
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

func SetTime(tm time.Time) Item {
	return func(form *Form) error {
		(*form).Time = tm
		return nil
	}
}

func SetService(str string) Item {
	return func(form *Form) error {
		(*form).Service = lib.SERVICE.ReadField(str, "service")
		return nil
	}
}

func SetAddress(str string) Item {
	return func(form *Form) error {
		(*form).Address = lib.SERVICE.ReadField(str, "address")
		return nil
	}
}

func SetDescription(str string) Item {
	return func(form *Form) error {
		(*form).Description = lib.SERVICE.ReadField(str, "description")
		return nil
	}
}

func SetSpecField(str string) Item {
	return func(form *Form) error {
		service := (*form).Service
		if len(service) > 0 {
			(*form).SpecField = lib.SERVICE.ReadSpecField(str, service)
			return nil
		}
		return errors.New("cannot set specfield without service")
	}
}

func SetPubkey(str string) Item {
	return func(form *Form) error {
		(*form).Pubkey = util.ReadPubKeyString(str)
		return nil
	}
}

func MakeForm(str string) (*Form, error) {
	form, err := NewForm(
		SetTime(time.Now().UTC()),
		SetService(str),
		SetAddress(str),
		SetDescription(str),
		SetSpecField(str),
		SetPubkey(str),
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
	return fmt.Sprintf("resolved %v", tm.String()[:16])
}

func ParseForm(form *Form) string {
	posted := (*form).Time.String()[:16] // to the minute
	status := CheckStatus((*form).Resolved)
	field := lib.SERVICE.FieldOpts((*form).Service).Field
	return "<li>" + fmt.Sprintf(line, "posted", posted) + fmt.Sprintf(line, "issue", (*form).Service) + fmt.Sprintf(line, "address", (*form).Address) + fmt.Sprintf(line, "description", (*form).Description) + fmt.Sprintf(line, field, (*form).SpecField) + fmt.Sprintf(line, "pubkey", (*form).Pubkey) + fmt.Sprintf(line, "status", status) + "</li>"
}

func FormID(form *Form) string {
	bytes := make([]byte, 32) // 64?
	yr, wk := (*form).Time.ISOWeek()
	items := []string{
		(*form).Service,
		(*form).Address,
		fmt.Sprintf("%d %d", yr, wk),
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
	before := lib.SERVICE.ReadField(str, "before")
	if len(before) > 0 {
		beforeDate := util.ParseDateString(before)
		if !((*form).Time.Before(beforeDate)) {
			return false
		}
	}
	after := lib.SERVICE.ReadField(str, "after")
	if len(after) > 0 {
		afterDate := util.ParseDateString(after)
		if !((*form).Time.After(afterDate)) {
			return false
		}
	}
	service := lib.SERVICE.ReadField(str, "service")
	if len(service) > 0 {
		if !(service == (*form).Service) {
			return false
		}
	}
	address := lib.SERVICE.ReadField(str, "address")
	if len(address) > 0 {
		if !util.SubstringMatch(address, (*form).Address) {
			return false
		}
	}
	specfield := lib.SERVICE.ReadSpecField(str, service)
	if len(specfield) > 0 {
		if !(specfield == (*form).SpecField) {
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
