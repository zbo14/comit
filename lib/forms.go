package lib

import (
	"bytes"
	"errors"
	"fmt"
	// "github.com/tendermint/go-wire"
	. "github.com/zballs/3ii/util"
	"time"
)

const (
	ErrMakeForm            = 100
	ErrFindForm            = 101
	ErrFormAlreadyResolved = 102
	field                  = "<strong style='opacity:0.8;'>%v</strong> <small>%v</small>" + "<br>"
)

type Form struct {
	Posted      string
	Issue       string
	Location    string
	Description string
	// Detail      string

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

func setIssue(str string) Item {
	return func(form *Form) error {
		form.Issue = ReadField(str, "issue")
		return nil
	}
}

func setLocation(str string) Item {
	return func(form *Form) error {
		form.Location = ReadField(str, "location")
		return nil
	}
}

func setDescription(str string) Item {
	return func(form *Form) error {
		form.Description = ReadField(str, "description")
		return nil
	}
}

/*
func setDetail(str string) Item {
	return func(form *Form) error {
		issue := form.Issue
		if len(issue) > 0 {
			form.Detail = ReadDetail(str, issue)
			return nil
		}
		return errors.New("cannot set form detail without issue")
	}
}
*/

func MakeForm(str string) (*Form, error) {
	timestr := time.Now().UTC().String()
	form, err := NewForm(
		setPosted(timestr),
		setIssue(str),
		setLocation(str),
		setDescription(str))
	if err != nil {
		return nil, err
	}
	form.Status = "unresolved"
	return form, nil
}

func XOR(bytes []byte, items ...string) []byte {
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

func (form *Form) ID() []byte {
	bytes := make([]byte, 16)
	daystr := ToTheDay(form.Posted)
	bytes = XOR(bytes, daystr, form.Issue) //form.Location
	return bytes
}

func (form *Form) Summary() string {
	status := "unresolved"
	if form.Resolved() {
		status = fmt.Sprintf(
			"resolved at %v by %v",
			form.ResolvedAt,
			form.ResolvedBy)
	}
	posted := ToTheMinute(form.Posted)
	/*
		sd := IssueDetail(form.Issue)
		detail := "no options"
		if sd != nil {
			detail = sd.Detail
		}
	*/
	var summary bytes.Buffer
	summary.WriteString("<li>" + fmt.Sprintf(field, "posted", posted))
	summary.WriteString(fmt.Sprintf(field, "issue", form.Issue))
	summary.WriteString(fmt.Sprintf(field, "location", form.Location))
	summary.WriteString(fmt.Sprintf(field, "description", form.Description))
	// summary.WriteString(fmt.Sprintf(field, detail, form.Detail))
	summary.WriteString(fmt.Sprintf(field, "status", status) + "<br></li>")
	return summary.String()
}

func MatchForm(str string, form *Form) bool {
	before := ReadField(str, "before")
	if len(before) > 0 {
		postedDate := ParseTimeString(form.Posted)
		beforeDate := ParseTimeString(before)
		if !(postedDate.Before(beforeDate)) {
			return false
		}
	}
	after := ReadField(str, "after")
	if len(after) > 0 {
		postedDate := ParseTimeString(form.Posted)
		afterDate := ParseTimeString(after)
		if !(postedDate.After(afterDate)) {
			return false
		}
	}
	issue := ReadField(str, "issue")
	if len(issue) > 0 {
		if !(issue == form.Issue) {
			return false
		}
	}
	location := ReadField(str, "location")
	if len(location) > 0 {
		if !SubstringMatch(location, form.Location) {
			return false
		}
	}
	status := ReadField(str, "status")
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
