package forms

import (
	. "github.com/zballs/comit/util"
)

const (
	ErrDecodeForm          = 100
	ErrFindForm            = 101
	ErrFormAlreadyResolved = 102
	ErrDecodeFormID        = 103
)

type Form struct {
	ContentType string `json:"content_type"`
	Data        []byte `json:"data, omitempty"`
	Description string `json:"description"`
	Issue       string `json:"issue"`
	Location    string `json:"location"`
	SubmittedAt string `json:"submitted_at"`
	Submitter   string `json:"submitter"`
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

func (form Form) ID() []byte {

	bytes := make([]byte, 16)

	minstr := ToTheMinute(form.SubmittedAt)

	bytes = XOR(bytes, minstr, form.Issue, form.Location)

	return bytes
}

/*
// TODO: add more file extensions

var fileTypes = map[string]string{
	"jpg": "image",
	"mp3": "audio",
	"mp4": "video",
}

// Media

type Media struct {
	Data     []byte
	Mimetype string `json:"mimetype"`
	Size     int    `json:"size"`

	// Content IDs
	TextID string `json:"text_id, omitempty"`
	NumID  int    `json:"num_id, omitempty"`
}

func newMedia(data []byte, fileType string, extension string) *Media {

	mimetype := fileType + "/" + extension

	return &Media{
		Data:     data,
		Mimetype: mimetype,
		Size:     len(data),
	}
}

type Form struct {

	// Required
	SubmittedAt string `json:"submitted-at"`
	Issue       string `json:"issue"`
	Location    string `json:"location"`
	Description string `json:"description"`

	// Optional
	Media     *Media `json:"media,omitempty"`
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

// TODO: add location
func (form *Form) ID() []byte {

	bytes := make([]byte, 16)

	daystr := ToTheMinute(form.SubmittedAt)

	bytes = XOR(bytes, daystr, form.Issue)

	return bytes
}

func (form *Form) SetContentIDs(textID string, numID int) {

	if form.Media == nil {
		return
	}

	form.Media.TextID = textID
	form.Media.NumID = numID
}

func (form *Form) MediaData() []byte {

	if form.Media == nil {
		return nil
	}

	return form.Media.Data
}

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
