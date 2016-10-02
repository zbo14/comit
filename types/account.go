package types

import (
	"fmt"
	"github.com/tendermint/go-crypto"
)

type Account struct {
	PubKey   crypto.PubKey
	Sequence int
}

func NewAccount(addr []byte, seq int) *Account {
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], addr[:])
	return &Account{
		PubKey:   pubKey,
		Sequence: seq,
	}
}

func (acc *Account) Copy() *Account {
	accCopy := *acc
	return &accCopy
}

func (acc *Account) String() string {
	if acc == nil {
		return "nil-Account"
	}
	return fmt.Sprintf("Account {%v %v}",
		acc.PubKey, acc.Sequence)
}

type PrivAccount struct {
	crypto.PrivKey
	Account
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
