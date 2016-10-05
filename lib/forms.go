package lib

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/zballs/3ii/util"
	"time"
)

const (
	line = "<strong style='opacity:0.8;'>%v</strong> <small>%v</small>" + "<br>"
)

type Form struct {
	Posted      string
	Service     string
	Address     string
	Description string
	Detail      string

	//==============//

	Resolved     string
	ResponseTime float64 `wire:"unsafe"`
}

// Functional Options

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

func setPosted(timestr string) Item {
	return func(form *Form) error {
		(*form).Posted = timestr
		return nil
	}
}

func setService(str string) Item {
	return func(form *Form) error {
		(*form).Service = SERVICE.ReadField(str, "service")
		return nil
	}
}

func setAddress(str string) Item {
	return func(form *Form) error {
		(*form).Address = SERVICE.ReadField(str, "address")
		return nil
	}
}

func setDescription(str string) Item {
	return func(form *Form) error {
		(*form).Description = SERVICE.ReadField(str, "description")
		return nil
	}
}

func setDetail(str string) Item {
	return func(form *Form) error {
		service := (*form).Service
		if len(service) > 0 {
			(*form).Detail = SERVICE.ReadDetail(str, service)
			return nil
		}
		return errors.New("cannot set form detail without service")
	}
}

func MakeForm(str string) (*Form, error) {
	timestr := time.Now().UTC().String()
	form, err := NewForm(
		setPosted(timestr),
		setService(str),
		setAddress(str),
		setDescription(str),
		setDetail(str),
	)
	if err != nil {
		return nil, err
	}
	return form, nil
}

func CheckStatus(timestr string) string {
	if len(timestr) == 0 {
		return "unresolved"
	}
	return fmt.Sprintf("resolved %v", ToTheMinute(timestr))
}

func (form *Form) ID() []byte {
	bytes := make([]byte, 16)
	daystr := ToTheDay(form.Posted)
	items := []string{
		daystr,
		(*form).Service,
		(*form).Address,
		(*form).Resolved,
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
	return bytes
}

func (form *Form) Summary() string {
	posted := ToTheMinute(form.Posted)
	status := CheckStatus(form.Resolved)
	sd := SERVICE.ServiceDetail(form.Service)
	detail := "no options"
	if sd != nil {
		detail = sd.Detail
	}
	var summary bytes.Buffer
	summary.WriteString("<li>" + fmt.Sprintf(line, "posted", posted))
	summary.WriteString(fmt.Sprintf(line, "service", form.Service))
	summary.WriteString(fmt.Sprintf(line, "address", form.Address))
	summary.WriteString(fmt.Sprintf(line, "description", form.Description))
	summary.WriteString(fmt.Sprintf(line, "description", form.Description))
	summary.WriteString(fmt.Sprintf(line, detail, form.Detail))
	summary.WriteString(fmt.Sprintf(line, "status", status) + "<br></li>")
	return summary.String()
}

/*
XXX TODO update
func MatchForm(str string, form *Form) bool {
	before := SERVICE.ReadField(str, "before")
	if len(before) > 0 {
		beforeDate := ParseTimeString(before)
		if !((*form).Posted().Before(beforeDate)) {
			return false
		}
	}
	after := SERVICE.ReadField(str, "after")
	if len(after) > 0 {
		afterDate := ParseTimeString(after)
		if !((*form).Posted().After(afterDate)) {
			return false
		}
	}
	service := SERVICE.ReadField(str, "service")
	if len(service) > 0 {
		if !(service == (*form).Service()) {
			return false
		}
	}
	address := SERVICE.ReadField(str, "address")
	if len(address) > 0 {
		if !SubstringMatch(address, (*form).Address()) {
			return false
		}
	}
	detail := SERVICE.ReadDetail(str, service)
	if len(detail) > 0 {
		if !(detail == (*form).Detail()) {
			return false
		}
	}
	return true
}
*/

//=========================================//

func (form *Form) Resolve(timestr string) error {
	if len(form.Resolved) > 0 {
		return errors.New("form already resolved")
	}
	(*form).Resolved = timestr
	(*form).ResponseTime = DurationHours(
		form.Posted, form.Resolved)
	return nil
}

// Errors

const (
	ErrMakeForm            = 311
	ErrFindForm            = 3111
	ErrFormAlreadyResolved = 31111
)
