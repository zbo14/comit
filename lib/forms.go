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

	Status     string
	ResolvedBy string
	ResolvedAt string
	// ResponseTime float64 `wire:"unsafe"`
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
		form.Posted = timestr
		return nil
	}
}

func setService(str string) Item {
	return func(form *Form) error {
		form.Service = SERVICE.ReadField(str, "service")
		return nil
	}
}

func setAddress(str string) Item {
	return func(form *Form) error {
		form.Address = SERVICE.ReadField(str, "address")
		return nil
	}
}

func setDescription(str string) Item {
	return func(form *Form) error {
		form.Description = SERVICE.ReadField(str, "description")
		return nil
	}
}

func setDetail(str string) Item {
	return func(form *Form) error {
		service := form.Service
		if len(service) > 0 {
			form.Detail = SERVICE.ReadDetail(str, service)
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
		setDetail(str))
	if err != nil {
		return nil, err
	}
	form.Status = "unresolved"
	return form, nil
}

func (form *Form) ID() []byte {
	bytes := make([]byte, 16)
	daystr := ToTheDay(form.Posted)
	items := []string{
		daystr,
		form.Service,
		form.Address,
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
	if form.Resolved() {
		status := fmt.Sprintf(
			"resolved at %v by %v",
			form.ResolvedAt,
			form.ResolvedBy)
	} else {
		status := "unresolved"
	}
	posted := ToTheMinute(form.Posted)
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
	summary.WriteString(fmt.Sprintf(line, detail, form.Detail))
	summary.WriteString(fmt.Sprintf(line, "status", status) + "<br></li>")
	return summary.String()
}

func MatchForm(str string, form *Form) bool {
	before := SERVICE.ReadField(str, "before")
	if len(before) > 0 {
		postedDate := ParseTimeString(form.Posted)
		beforeDate := ParseTimeString(before)
		if !(postedDate.Before(beforeDate)) {
			return false
		}
	}
	after := SERVICE.ReadField(str, "after")
	if len(after) > 0 {
		postedDate := ParseTimeString(form.Posted)
		afterDate := ParseTimeString(after)
		if !(postedDate.After(afterDate)) {
			return false
		}
	}
	service := SERVICE.ReadField(str, "service")
	if len(service) > 0 {
		if !(service == form.Service) {
			return false
		}
	}
	address := SERVICE.ReadField(str, "address")
	if len(address) > 0 {
		if !SubstringMatch(address, form.Address) {
			return false
		}
	}
	status := SERVICE.ReadField(str, "status")
	if len(status) > 0 {
		if !(status == form.Status) {
			return false
		}
	}
	return true
}

//=========================================//

func (form *Form) Resolved() bool {
	return form.Status == "resolved"
}

func (form *Form) Resolve(timestr, pubKeyString string) error {
	if form.Resolved() {
		return errors.New("form already resolved")
	}
	form.Status = "resolved"
	form.ResolvedAt = timestr
	form.ResolvedBy = pubKeyString
	return nil
}

// Errors

const (
	ErrMakeForm            = 311
	ErrFindForm            = 3111
	ErrFormAlreadyResolved = 31111
)
