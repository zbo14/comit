package types

import (
	"fmt"
	. "github.com/zballs/comit/util"
	"gx/ipfs/QmcEcrBAMrwMyhSjXt4yfyPpzgSuV8HLHavnfmiKCSRqZU/go-cid"
)

const FORM_ID_LENGTH = 16

// Info contains the id pair for a submitted form,
// fields relevant to state filters (issue, location),
// and submitter so we know when to send a receipt

type Info struct {
	ContentID *cid.Cid `json:"content_id"`
	FormID    []byte   `json:"form_id"`
	Issue     string   `json:"issue"`
	Location  string   `json:"location"`
	Submitter string   `json:"submitter"`
}

func NewInfo(contentID *cid.Cid, form Form) Info {
	return Info{contentID, form.ID(), form.Issue, form.Location, form.Submitter}
}

type Form struct {
	ContentType string `json:"content_type, omitempty"`
	Data        []byte `json:"data, omitempty"`
	Description string `json:"description"`
	Issue       string `json:"issue"`
	Location    string `json:"location"`
	SubmittedAt string `json:"submitted_at"`
	Submitter   string `json:"submitter"`
}

func xor(bytes []byte, items ...string) []byte {
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

func (form Form) ID() []byte {
	bytes := make([]byte, 16)
	minutestr := ToTheMinute(form.SubmittedAt)
	bytes = xor(bytes, minutestr, form.Issue, form.Location)
	return bytes
}

func (form Form) String() string {
	return form.StringIndented("")
}

func (form Form) StringIndented(indent string) string {
	return fmt.Sprintf(`Form{
	%s ContentType: %v
	%s DataSize: %v
	%s Description: %v 
	%s Issue: %v
	%s Location: %v 
	%s SubmittedAt: %v 
	%s Submitter: %v
	}`,
		indent, form.ContentType,
		indent, len(form.Data),
		indent, form.Description,
		indent, form.Issue,
		indent, form.Location,
		indent, form.SubmittedAt,
		indent, form.Submitter)
}
