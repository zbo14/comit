package types

import (
	"github.com/pkg/errors"
	. "github.com/zballs/comit/util"
)

type Update struct {
	Error   error    `json:"error, omitempty"`
	Form    *Form    `json:"form, omitempty"`
	Receipt *Receipt `json:"receipt, omitempty"`
	Type    string   `json:"type"`
}

func NewUpdate(v interface{}, err error) (*Update, error) {
	switch v.(type) {
	case *Form:
		return &Update{
			Error: err,
			Form:  v.(*Form),
			Type:  "form",
		}, nil
	case *Receipt:
		return &Update{
			Error:   err,
			Receipt: v.(*Receipt),
			Type:    "receipt",
		}, nil
	default:
		return nil, errors.New("Unrecognized update type")
	}
}

type Receipt struct {
	AppHash     []byte `json:"app_hash"`
	BlockHeight int    `json:"block_height"`
	FormID      string `json:"form_id"`
}

func NewReceipt(blockHeight int, formID []byte) *Receipt {
	return &Receipt{BlockHeight: blockHeight, FormID: BytesToHexstr(formID)}
}
