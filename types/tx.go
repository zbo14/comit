package types

import (
	"encoding/json"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	CreateAccountTx = 0x01
	RemoveAccountTx = 0x02
	CreateAdminTx   = 0x03
	RemoveAdminTx   = 0x04
	SubmitTx        = 0x05
	ResolveTx       = 0x06
	//CheckTxs - do not change state
	ConnectTx = 0x07
	UpdateTx  = 0x08

	// Errors
	ErrUnexpectedData       = 1000
	ErrAccountAlreadyExists = 2000
)

type TxInput struct {
	Address   []byte           `json: "address"`
	Sequence  int              `json: "sequence"`
	Signature crypto.Signature `json: "signature"`
	PubKey    crypto.PubKey    `json: "public-key"`
}

func (txIn TxInput) ValidateBasic() tmsp.Result {
	if len(txIn.Address) != 20 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Invalid address length")
	}
	if txIn.Sequence <= 0 {
		return tmsp.ErrBaseInvalidInput.AppendLog("Sequence must be greater than 0")
	}
	if txIn.Sequence == 1 && txIn.PubKey == nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be present when Sequence == 1")
	}
	if txIn.Sequence > 1 && txIn.PubKey != nil {
		return tmsp.ErrBaseInvalidInput.AppendLog("PubKey must be nil when Sequence > 1")
	}
	return tmsp.OK
}

func (txIn TxInput) String() string {
	return fmt.Sprintf("TxInput{%X,%v,%v,%v}", txIn.Address, txIn.Sequence, txIn.Signature, txIn.PubKey)
}

type Tx struct {
	Type  byte    `json: "type"`
	Input TxInput `json: "input"`
	Data  []byte  `json: "data"`
}

func (tx *Tx) SignBytes(chainID string) []byte {
	signBytes := wire.BinaryBytes(chainID)
	sig := tx.Input.Signature
	tx.Input.Signature = nil
	signBytes = append(signBytes, wire.BinaryBytes(tx)...)
	tx.Input.Signature = sig
	return signBytes
}

func (tx *Tx) SetSequence(seq int) {
	tx.Input.Sequence = seq + 1
}

func (tx *Tx) SetAccount(pubKey crypto.PubKey) {
	if tx.Input.Sequence == 1 {
		tx.Input.PubKey = pubKey
	}
	tx.Input.Address = pubKey.Address()
}

func (tx *Tx) SetSignature(privKey crypto.PrivKey, chainID string) {
	tx.Input.Signature = privKey.Sign(tx.SignBytes(chainID))
}

func (tx *Tx) String() string {
	return fmt.Sprintf("Tx{%v %v %X}", tx.Type, tx.Input, tx.Data)
}

func TxID(chainID string, tx Tx) []byte {
	signBytes := tx.SignBytes(chainID)
	return wire.BinaryRipemd160(signBytes)
}

func jsonEscape(str string) string {
	escapedBytes, err := json.Marshal(str)
	if err != nil {
		PanicSanity(Fmt("Error json-escaping a string", str))
	}
	return string(escapedBytes)
}
