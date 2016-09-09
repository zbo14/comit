package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	types "github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/util"
)

// Account

type Account struct {
	PubKeyEd25519
}

func (a Account) Sign() []byte {
	return []byte(fmt.Sprintf("PubKeyEd25519{%X}", a.Bytes()))
}

func (a Account) SubmitForm(tx []byte, app *Application) types.Result {
	tx = append(tx, a.Sign()...)
	return app.AppendTx(tx)
}

func (Account) QueryForm(tx []byte, cache *Cache) *Form {
	return (*cache).QueryForm(string(tx))
}

func (Account) QueryResolved(tx []byte, cache *Cache) *Form {
	return (*cache).QueryResolved(string(tx))
}

// Accounts, Accountdb

type Accounts map[string]*Account
type Accountdb chan Accounts

func CreateAccountdb() Accountdb {
	return make(chan Accounts, 1)
}

func (db Accountdb) Access() Accounts {
	return <-db
}

func (db Accountdb) Return(accounts Accounts, done chan struct{}) {
	db <- accounts
	done <- struct{}{}
}

func (db Accountdb) Add(privkey PrivKeyEd25519, account *Account) error {
	accounts := db.Access()
	if accounts[string(privkey[:])] != nil {
		return errors.New("account with private key already exists")
	}
	accounts[string(privkey[:])] = account
	done := make(chan struct{}, 1)
	go db.Return(accounts, done)
	select {
	case <-done:
		return nil
	}
}

func (db Accountdb) Remove(privkey PrivKeyEd25519) error {
	accounts := db.Access()
	var err error
	if accounts[string(privkey[:])] != nil {
		delete(accounts, string(privkey[:]))
		err = nil
	} else {
		err = errors.New("account with private key does not exist")
	}
	done := make(chan struct{}, 1)
	go db.Return(accounts, done)
	select {
	case <-done:
		return err
	}
}

// Account Manager

type AccountManager struct {
	Accountdb
}

func CreateAccountManager() *AccountManager {
	return &AccountManager{CreateAccountdb()}
}

func (am AccountManager) CreateAccount(secret []byte) (PrivKeyEd25519, error) {
	account := Account{}
	privkey := GenPrivKeyEd25519FromSecret(secret)
	copy(account.Bytes(), privkey.PubKey().Bytes())
	err := am.Add(privkey, &account)
	if err != nil {
		for idx, _ := range privkey {
			privkey[idx] = byte(0)
		}
	}
	return privkey, err
}

func (am AccountManager) RemoveAccount(privkey PrivKeyEd25519) error {
	return am.Remove(privkey)
}

func (am AccountManager) SubmitForm(tx []byte, app *Application) types.Result {
	accounts := am.Access()
	privkey := ExtractPrivKeyEd25519(tx)
	accountPtr := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go am.Return(accounts, done)
	select {
	case <-done:
		if accountPtr == nil {
			return types.NewResult(types.CodeType_InternalError, nil, "account with private key does not exist")
		}
		return (*accountPtr).SubmitForm(tx, app)
	}
}

func (am AccountManager) QueryForm(tx []byte, cache *Cache) *Form {
	accounts := am.Access()
	privkey := ExtractPrivKeyEd25519(tx)
	accountPtr := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go am.Return(accounts, done)
	select {
	case <-done:
		if accountPtr == nil {
			return nil
		}
		return (*accountPtr).QueryForm(tx, cache)
	}
}

func (am AccountManager) QueryResolved(tx []byte, cache *Cache) *Form {
	accounts := am.Access()
	privkey := ExtractPrivKeyEd25519(tx)
	accountPtr := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go am.Return(accounts, done)
	select {
	case <-done:
		if accountPtr == nil {
			return nil
		}
		return (*accountPtr).QueryResolved(tx, cache)
	}
}
