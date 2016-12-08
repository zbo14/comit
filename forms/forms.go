package forms

import (
	"errors"
	"fmt"
	. "github.com/zballs/comit/util"
	"time"
)

var Fmt = fmt.Sprintf

const (
	ErrDecodeForm          = 100
	ErrFindForm            = 101
	ErrFormAlreadyResolved = 102
	ErrDecodingFormID      = 103
)

// TODO: add more file extensions

var fileTypes = map[string]string{
	"jpg": "image",
	"mp3": "audio",
	"mp4": "video",
}

// Media

type Media struct {
	Data      []byte `json:"data"`
	Type      string `json:"type"`
	Extension string `json:"extension"`

	// Content IDs
	TextID string `json:"text_id, omitempty"`
	NumID  int    `json:"num_id, omitempty"`
}

func newMedia(data []byte, fileType string, extension string) *Media {

	return &Media{
		Data:      data,
		Type:      fileType,
		Extension: extension,
	}
}

type Form struct {

	// Required
	SubmittedAt string `json:"submitted-at"`
	Issue       string `json:"issue"`
	Location    string `json:"location"`
	Description string `json:"description"`

	// Optional
	*Media    `json:"media,omitempty"`
	Submitter string `json:"submitter,omitempty"`

	// -------------------------------- //

	Status     string `json:"status"`
	ResolvedBy string `json:"resolved-by,omitempty"`
	ResolvedAt string `json:"resolved-at,omitempty"`
}

// Functional Options

type Item func(*Form) error

func newForm(items ...Item) (*Form, error) {
	form := &Form{}
	for _, item := range items {
		err := item(form)
		if err != nil {
			return nil, err
		}
	}
	return form, nil
}

func setSubmittedAt(submittedAt string) Item {
	return func(form *Form) error {
		form.SubmittedAt = submittedAt
		return nil
	}
}

func setIssue(issue string) Item {
	return func(form *Form) error {
		form.Issue = issue
		return nil
	}
}

func setLocation(location string) Item {
	return func(form *Form) error {
		form.Location = location
		return nil
	}
}

func setDescription(description string) Item {
	return func(form *Form) error {
		form.Description = description
		return nil
	}
}

func setMedia(data []byte, extension string) Item {

	return func(form *Form) error {

		if len(data) < 50 {
			return nil
		}

		fileType, ok := fileTypes[extension]

		if !ok {
			return errors.New("Unrecognized file extension")
		}

		fmt.Printf("Media size: %d\n", len(data))

		form.Media = newMedia(data, fileType, extension)

		return nil
	}
}

func setSubmitter(submitter string) Item {
	return func(form *Form) error {
		form.Submitter = submitter
		return nil
	}
}

func MakeForm(issue, location, description string, data []byte, extension, submitter string) (*Form, error) {

	submittedAt := time.Now().Local().String()

	form, err := newForm(
		setSubmittedAt(submittedAt),
		setIssue(issue),
		setLocation(location),
		setDescription(description),
		setMedia(data, extension),
		setSubmitter(submitter))

	if err != nil {
		return nil, err
	}

	form.Status = "unresolved"

	return form, nil
}

func (form *Form) Resolved() bool {
	return form.Status == "resolved"
}

func (form *Form) Resolve(timestr, addr string) error {

	if form.Resolved() {
		return errors.New("Form already resolved")
	}

	form.Status = "resolved"

	form.ResolvedAt = timestr
	form.ResolvedBy = addr

	return nil
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

func (form *Form) HasMedia() bool {
	return form.Media != nil
}

// TODO: add location
func (form *Form) ID() []byte {

	bytes := make([]byte, 16)

	daystr := ToTheMinute(form.SubmittedAt)

	bytes = XOR(bytes, daystr, form.Issue)

	return bytes
}

func (form *Form) SetContentIDs(textID string, numID int) {

	if !form.HasMedia() {
		return
	}

	form.Media.TextID = textID
	form.Media.NumID = numID
}

/*
	if !form.HasMedia() {
		return
	}

	var content bytes.Buffer

	b64 := base64.StdEncoding.EncodeToString(form.Media)

	switch fileTypes[form.Extension] {
	case "image":
		content.WriteString(Fmt(imageElement, textID, numID, form.Extension))
	case "audio":
		content.WriteString(Fmt(audioElement, textID, numID, form.Extension))
	case "video":
		content.WriteString(Fmt(videoElement, textID, numID, form.Extension))
	}

	content.WriteString(Fmt(mediaScript, b64, form.Extension, textID, numID))

	form.Content = content.String()

	form.Media = nil
	form.Extension = ""

*/

/*
func (form *Form) MediaDecomp() []byte {

	r := bytes.NewReader(form.Media)

	flate.NewRE

	decompressor := flate.NewReader(r)
	defer decompressor.Close()

	decompressed := make([]byte, form.Size)

	read := 0

	for {

		n, _ := decompressor.Read(decompressed[read:])
		read += n

		if read >= form.Size {
			break
		}
	}

	fmt.Printf("Decompressed size = %d\n", read)

	return decompressed
}
*/
