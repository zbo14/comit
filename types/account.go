package types

import (
	"fmt"
	"github.com/tendermint/go-crypto"
)

type Account struct {
	PubKey     crypto.PubKey `json:"pub_key"`
	Sequence   int           `json:"sequence"`
	permission int           `json:"permission"`
}

func NewAccount(pubKey crypto.PubKey, permission int) *Account {
	return &Account{
		PubKey:     pubKey,
		Sequence:   0,
		permission: permission,
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
	return fmt.Sprintf("Account {%X %v}",
		acc.PubKey, acc.Sequence)
}

func (acc *Account) PermissionToResolve() bool {
	return acc.permission >= 1
}

func (acc *Account) PermissionToCreateAdmin() bool {
	return acc.permission >= 2
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
