package types

import "github.com/tendermint/go-crypto"
import . "github.com/zballs/comit/util"

type Account struct {
	FormIDs  []string      `json:"form_ids"`
	PubKey   crypto.PubKey `json:"pub_key"`
	Sequence int           `json:"sequence"`
	Username string        `json:"username"`
}

func NewAccount(pubKey crypto.PubKey, username string) *Account {
	return &Account{
		PubKey:   pubKey,
		Sequence: 0,
		Username: username,
	}
}

func (acc *Account) Addform(form Form) {
	formID := BytesToHexstr(form.ID())
	acc.FormIDs = append(acc.FormIDs, formID)
}

func (acc *Account) Copy() *Account {
	return &*acc
}

type PrivAccount struct {
	*Account
	PrivKey crypto.PrivKey
}

func NewPrivAccount(acc *Account, privKey crypto.PrivKey) *PrivAccount {
	return &PrivAccount{acc, privKey}
}

type AccountGetter interface {
	GetAccount(addr []byte) *Account
}

type AccountSetter interface {
	SetAccount(addr []byte, acc *Account)
}

type AccountGetterSetter interface {
	GetAccount(addr []byte) *Account
	SetAccount(addr []byte, acc *Account)
}
