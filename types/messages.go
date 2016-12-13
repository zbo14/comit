package types

import (
	"github.com/tendermint/go-crypto"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	. "github.com/zballs/comit/util"
)

type Message struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data, omitempty"`
	Error  error       `json:"error, omitempty"`
	Result *Result     `json:"result, omitempty"`
}

type Result struct {
	Log string `json:"log"`
	Ok  bool   `json:"ok"`
}

func NewResult(log string, code int) *Result {
	return &Result{log, code == 0}
}

type Keypair struct {
	PrivKeystr string `json:"priv_key"`
	PubKeystr  string `json:"pub_key"`
}

func NewKeypair(pubKey crypto.PubKey, privKey crypto.PrivKey) *Keypair {
	return &Keypair{PrivKeytoHexstr(privKey), PubKeytoHexstr(pubKey)}
}

func MessageChainID(err error, result *ctypes.ResultTMSPQuery) *Message {
	tmResult := result.Result
	return &Message{
		Action: "chain_id",
		Error:  err,
		Result: NewResult(tmResult.Log, int(tmResult.Code)),
	}
}

func MessageIssues(err error, data []string, result *ctypes.ResultTMSPQuery) *Message {
	tmResult := result.Result
	return &Message{
		Action: "issues",
		Error:  err,
		Data:   data,
		Result: NewResult(tmResult.Log, int(tmResult.Code)),
	}
}

func MessageLogin(err error, result *ctypes.ResultTMSPQuery) *Message {
	tmResult := result.Result
	return &Message{
		Action: "login",
		Error:  err,
		Result: NewResult(tmResult.Log, int(tmResult.Code)),
	}
}

func MessageCreateAccount(err error, data *Keypair, result *ctypes.ResultBroadcastTx) *Message {
	return &Message{
		Action: "create_account",
		Error:  err,
		Data:   data,
		Result: NewResult(result.Log, int(result.Code)),
	}
}

func MessageRemoveAccount(err error, result *ctypes.ResultBroadcastTx) *Message {
	return &Message{
		Action: "remove_account",
		Error:  err,
		Result: NewResult(result.Log, int(result.Code)),
	}
}

func MessageSubmitForm(err error, formID []byte, result *ctypes.ResultBroadcastTx) *Message {
	m := &Message{
		Action: "submit_form",
		Error:  err,
		Result: NewResult(result.Log, int(result.Code)),
	}
	if formID != nil {
		m.Data = BytesToHexstr(formID)
	}
	return m
}

func MessageFindForm(err error, form *Form, result *ctypes.ResultTMSPQuery) *Message {
	tmResult := result.Result
	return &Message{
		Action: "find_form",
		Error:  err,
		Data:   form,
		Result: NewResult(tmResult.Log, int(tmResult.Code)),
	}
}
