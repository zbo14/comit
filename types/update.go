package types

import (
	"github.com/pkg/errors"
	tndr "github.com/tendermint/tendermint/types"
)

const UpdateForm string = "form"
const UpdateReceipt string = "receipt"

type Update struct {
	Type     string `json:"type"`
	*Form    `json:"form, omitempty"`
	*Receipt `json:"receipt, omitempty"`
}

func NewUpdate(v interface{}) (*Update, error) {
	switch v.(type) {
	case *Form:
		return &Update{
			Type: UpdateForm,
			Form: v.(*Form),
		}, nil
	case *Receipt:
		return &Update{
			Type:    UpdateReceipt,
			Receipt: v.(*Receipt),
		}, nil
	default:
		return nil, errors.New("Unrecognized update type")
	}
}

type Receipt struct {
	tndr.Tx     `json:"tx"`
	*tndr.Block `json:"block"`
}

func NewReceipt(tx tndr.Tx, block *tndr.Block) *Receipt {
	return &Receipt{tx, block}
}
