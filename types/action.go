package types

import (
	"fmt"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	tndr "github.com/tendermint/tendermint/types"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	ActionCreateAccount = 0x01
	ActionRemoveAccount = 0x02
	ActionSubmitForm    = 0x03
)

type ActionInput struct {
	Address   []byte           `json: "address"`
	Sequence  int              `json: "sequence"`
	Signature crypto.Signature `json: "signature"`
	PubKey    crypto.PubKey    `json: "public-key"`
}

func (in ActionInput) ValidateBasic() tmsp.Result {
	if len(in.Address) != 20 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Invalid address length")
	}
	if in.Sequence <= 0 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Sequence must be greater than 0")
	}
	if in.Sequence == 1 && in.PubKey == nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be present when Sequence == 1")
	}
	if in.Sequence > 1 && in.PubKey != nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be nil when Sequence > 1")
	}
	return tmsp.OK
}

func (in ActionInput) StringIndented(indent string) string {
	return fmt.Sprintf(`Input{
		%s %s Address: %x
		%s %s Sequence: %v
		%s %s PubKey: %v
		%s %s}`,
		indent, indent, in.Address,
		indent, indent, in.Sequence,
		indent, indent, in.PubKey,
		indent, indent)
}

func (in ActionInput) String() string {
	return in.StringIndented("")
}

type Action struct {
	Type  byte         `json: "type"`
	Input *ActionInput `json: "input"`
	Data  []byte       `json: "data"`
}

func NewAction(actionType byte, data []byte) Action {
	return Action{
		Type:  actionType,
		Input: &ActionInput{},
		Data:  data,
	}
}

func (a Action) SignBytes(chainID string) []byte {
	signBytes := wire.BinaryBytes(chainID)
	sig := a.Input.Signature
	a.Input.Signature = nil
	signBytes = append(signBytes, wire.BinaryBytes(a)...)
	a.Input.Signature = sig
	return signBytes
}

func (a Action) Prepare(pubKey crypto.PubKey, seq int) {
	a.Input.Sequence = seq
	if a.Input.Sequence == 1 {
		a.Input.PubKey = pubKey
	}
	a.Input.Address = pubKey.Address()
}

func (a Action) Sign(privKey crypto.PrivKey, chainID string) {
	a.Input.Signature = privKey.Sign(a.SignBytes(chainID))
}

func (a Action) ID(chainID string) []byte {
	signBytes := a.SignBytes(chainID)
	return wire.BinaryRipemd160(signBytes)
}

func (a Action) Tx() tndr.Tx {
	return wire.BinaryBytes(a)
}

func (a Action) StringIndented(indent string) string {
	return fmt.Sprintf(`Action{
		%s Type: %v
		%s %v 
		%s Data: %x
		}`,
		indent, a.Type,
		indent, a.Input,
		indent, a.Data,
		indent)
}

func (a Action) String() string {
	return a.StringIndented("")
}
