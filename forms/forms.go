package forms

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/base64"
	"errors"
	"fmt"
	. "github.com/zballs/comit/util"
	"io/ioutil"
	"time"
)

var Fmt = fmt.Sprintf

const (
	ErrDecodeForm          = 100
	ErrFindForm            = 101
	ErrFormAlreadyResolved = 102
	ErrDecodingFormID      = 103

	field     = ""
	miniField = ""

	imageElement = "<img width='240' height='240' id='%s-media-%d' name='%s'><br>"
	audioElement = "<audio id='%s-media-%d' name='%s' controls></audio><br>"
	videoElement = "<video width='240' height='240' id='%s-media-%d' name='%s' controls></video><br>"

	mediaScript = `<script type='text/javascript'>
					var byteChars = atob('%s'); 
					var byteNums = new Array(byteChars.length);
					for (var i = 0; i < byteNums.length; i++) {
						byteNums[i] = byteChars.charCodeAt(i);
					}
					var byteArray = new Uint8Array(byteNums);
					var blob = new Blob([byteArray], {type: '%s'});
					var media = document.getElementById('%s-media-%d');
					media.src = window.URL.createObjectURL(blob);
				  </script>`
)

var fileFormats = map[string]string{
	"jpg": "image",
	"mp3": "audio",
	"mp4": "video",
}

type Form struct {
	SubmittedAt string `json:"submitted-at"`
	Issue       string `json:"issue"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Media       []byte `json:"media"`     //optional
	Extension   string `json:"extension"` //optional
	Submitter   string `json:"submitter"` //optional

	// -------------------------------- //

	Status     string `json:"status"`
	ResolvedBy string `json:"resolved-by"`
	ResolvedAt string `json:"resolved-at"`
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

func setMedia(mediaBytes []byte, extension string) Item {
	compressed := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(compressed, 9)
	defer compressor.Close()
	compressor.Write(mediaBytes)
	return func(form *Form) error {
		form.Media = compressed.Bytes()
		form.Extension = extension
		return nil
	}
}

func setSubmitter(submitter string) Item {
	return func(form *Form) error {
		form.Submitter = submitter
		return nil
	}
}

func MakeAnonymousForm(issue, location, description string, mediaBytes []byte, extension string) (*Form, error) {
	submittedAt := time.Now().Local().String()
	form, err := newForm(
		setSubmittedAt(submittedAt),
		setIssue(issue),
		setLocation(location),
		setDescription(description))
	if err != nil {
		return nil, err
	}
	if mediaBytes != nil && len(extension) > 0 {
		err = setMedia(mediaBytes, extension)(form)
		if err != nil {
			return nil, err
		}
	}
	form.Status = "unresolved"
	return form, nil
}

func MakeForm(issue, location, description, submitter string, mediaBytes []byte, extension string) (*Form, error) {
	form, err := MakeAnonymousForm(issue, location, description, mediaBytes, extension)
	if err != nil {
		return nil, err
	}
	err = setSubmitter(submitter)(form)
	if err != nil {
		return nil, err
	}
	return form, nil
}

func (form *Form) Resolved() bool {
	return form.Status == "resolved"
}

func (form *Form) Resolve(timestr, addr string) error {
	if form.Resolved() {
		return errors.New("form already resolved")
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

func (form *Form) hasMedia() bool {
	if _, ok := fileFormats[form.Extension]; !ok {
		return false
	}
	return form.Media != nil
}

// TODO: add location
func (form *Form) ID() []byte {
	bytes := make([]byte, 16)
	daystr := ToTheMinute(form.SubmittedAt)
	bytes = XOR(bytes, daystr, form.Issue)
	return bytes
}

func (form *Form) Summary(textID string, numID int) string {
	status := "unresolved"
	if form.Resolved() {
		status = Fmt(
			"resolved at %v by <really-small>%v</really-small>",
			form.ResolvedAt,
			form.ResolvedBy)
	}
	submittedAt := ToTheMinute(form.SubmittedAt)
	var summary bytes.Buffer
	summary.WriteString(Fmt(field, "submitted", submittedAt))
	summary.WriteString(Fmt(field, "issue", form.Issue))
	summary.WriteString(Fmt(field, "location", form.Location))
	summary.WriteString(Fmt(field, "description", form.Description))
	summary.WriteString(Fmt(field, "status", status))
	if len(form.Submitter) == 64 {
		summary.WriteString(Fmt(miniField, "submitter", form.Submitter))
	}
	if form.hasMedia() {
		decomp, err := form.MediaDecomp()
		if err != nil {
			fmt.Println("Error: failed to decompress media bytes")
			return summary.String()
		}
		b64 := base64.StdEncoding.EncodeToString(decomp)
		switch fileFormats[form.Extension] {
		case "image":
			summary.WriteString(
				Fmt(imageElement, textID, numID, form.Extension))
		case "audio":
			summary.WriteString(
				Fmt(audioElement, textID, numID, form.Extension))
		case "video":
			summary.WriteString(
				Fmt(videoElement, textID, numID, form.Extension))
		}
		summary.WriteString(
			Fmt(mediaScript, b64, form.Extension, textID, numID))
	}
	return summary.String()
}

func (form *Form) MediaContent(textID string, numID int) ([]byte, error) {

	var content bytes.Buffer

	if !form.hasMedia() {
		return nil, errors.New("No media")
	}

	decomp, err := form.MediaDecomp()

	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(decomp)

	switch fileFormats[form.Extension] {
	case "image":
		content.WriteString(Fmt(imageElement, textID, numID, form.Extension))
	case "audio":
		content.WriteString(Fmt(audioElement, textID, numID, form.Extension))
	case "video":
		content.WriteString(Fmt(videoElement, textID, numID, form.Extension))
	}

	content.WriteString(Fmt(mediaScript, b64, form.Extension, textID, numID))

	return content.Bytes(), nil
}

func (form *Form) MediaDecomp() ([]byte, error) {
	r := new(bytes.Buffer)
	r.Write(form.Media)
	bufr := bufio.NewReader(r)
	decompressor := flate.NewReader(bufr)
	defer decompressor.Close()
	decompressed, err := ioutil.ReadAll(decompressor)
	if err != nil {
		return nil, err
	}
	return decompressed, nil
}
