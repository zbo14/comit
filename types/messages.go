package types

import (
	"github.com/pkg/errors"
	"github.com/tendermint/go-crypto"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	. "github.com/zballs/comit/util"
	"gx/ipfs/QmcEcrBAMrwMyhSjXt4yfyPpzgSuV8HLHavnfmiKCSRqZU/go-cid"
)

type Message struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data, omitempty"`
	Error  error       `json:"error, omitempty"`
}

type Keypair struct {
	PrivKeystr string `json:"priv_key"`
	PubKeystr  string `json:"pub_key"`
}

type Idpair struct {
	FormID    string `json:"form_id"`
	ContentID string `json:"content_id"`
}

func ResultToError(result interface{}) error {
	switch result.(type) {
	case *ctypes.ResultTMSPQuery:
		tmResult := result.(*ctypes.ResultTMSPQuery).Result
		if tmResult.Code == 0 {
			return nil
		}
		return errors.New(tmResult.Error())
	case *ctypes.ResultBroadcastTx:
		_result := result.(*ctypes.ResultBroadcastTx)
		if _result.Code == 0 {
			return nil
		}
		return errors.New(_result.Log)
	default:
		return errors.New("Unrecognized result type")
	}
}

func NewKeypair(pubKey crypto.PubKey, privKey crypto.PrivKey) *Keypair {
	return &Keypair{PrivKeytoHexstr(privKey), PubKeytoHexstr(pubKey)}
}

func NewIdpair(form Form, cid *cid.Cid) *Idpair {
	return &Idpair{BytesToHexstr(form.ID()), cid.String()}
}

func MessageChainID(err error) *Message {
	return &Message{
		Action: "chain_id",
		Error:  err,
	}
}

func MessageIssues(data []string, err error) *Message {
	return &Message{
		Action: "issues",
		Data:   data,
		Error:  err,
	}
}

func MessageLogin(err error) *Message {
	return &Message{
		Action: "login",
		Error:  err,
	}
}

func MessageCreateAccount(data *Keypair, err error) *Message {
	return &Message{
		Action: "create_account",
		Data:   data,
		Error:  err,
	}
}

func MessageRemoveAccount(err error) *Message {
	return &Message{
		Action: "remove_account",
		Error:  err,
	}
}

func MessageSubmitForm(data *Idpair, err error) *Message {
	return &Message{
		Action: "submit_form",
		Data:   data,
		Error:  err,
	}
}

func MessageFindForm(data *Form, err error) *Message {
	return &Message{
		Action: "find_form",
		Data:   data,
		Error:  err,
	}
}
