package types

import . "github.com/zballs/comit/util"

const FORM_ID_LENGTH = 16

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
