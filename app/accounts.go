package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	types "github.com/tendermint/tmsp/types"
	util "github.com/zballs/3ii/util"
)

// Account

type Account PubKeyEd25519

func (a *Account) ToString() string {
	return fmt.Sprintf("%x", a[:])
}

func (a *Account) Validate(pubKeyString string) bool {
	return pubKeyString == a.ToString()
}

func (a *Account) SubmitForm(str string, app *Application) types.Result {
	pubKeyString := util.ReadPubKeyString(str)
	if !a.Validate(pubKeyString) {
		return types.NewResult(types.CodeType_InternalError, nil, "invalid public-private key pair")
	}
	tx := []byte(str)
	return app.AppendTx(tx)
}

func (a *Account) QueryForm(str string, cache *Cache) (*Form, error) {
	pubKeyString := util.ReadPubKeyString(str)
	if !a.Validate(pubKeyString) {
		return nil, errors.New("invalid public-private key pair")
	}
	id := util.ReadFormID(str)
	return cache.QueryForm(id)
}

// Accounts, Accountdb

type Accounts map[string]*Account
type Accountdb chan Accounts

func CreateAccountdb() Accountdb {
	accountdb := make(chan Accounts, 1)
	done := make(chan struct{}, 1)
	go func() {
		accountdb <- Accounts{}
		done <- struct{}{}
	}()
	select {
	case <-done:
		return accountdb
	}
}

func (db *Accountdb) Access() Accounts {
	return <-(*db)
}

func (db *Accountdb) Return(accounts Accounts, done chan struct{}) {
	(*db) <- accounts
	done <- struct{}{}
}

func (db *Accountdb) Add(privkey PrivKeyEd25519, account *Account) error {
	accounts := db.Access()
	if accounts[util.PrivKeyToString(privkey)] != nil {
		return errors.New("account with private key already exists")
	}
	accounts[util.PrivKeyToString(privkey)] = account
	fmt.Println(accounts)
	done := make(chan struct{}, 1)
	go db.Return(accounts, done)
	select {
	case <-done:
		return nil
	}
}

func (db *Accountdb) Remove(pubKeyString string, privKeyString string) error {
	var err error = nil
	accounts := db.Access()
	account := accounts[privKeyString]
	if account != nil {
		if account.Validate(pubKeyString) {
			delete(accounts, privKeyString)
			err = nil
		} else {
			err = errors.New("invalid public-private key pair")
		}
	} else {
		err = errors.New("account with private key does not exist")
	}
	fmt.Println(accounts)
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

func (am *AccountManager) CreateAccount(passphrase string) (pubkey PubKeyEd25519, privkey PrivKeyEd25519, err error) {
	var account = Account{}
	secret := util.GenerateSecret([]byte(passphrase))
	privkey = GenPrivKeyEd25519FromSecret(secret)
	copy(pubkey[:], privkey.PubKey().Bytes())
	copy(account[:], pubkey[:])
	err = am.Add(privkey, &account)
	return pubkey, privkey, err
}

func (am *AccountManager) RemoveAccount(pubKeyString string, privKeyString string) error {
	return am.Remove(pubKeyString, privKeyString)
}

func (am *AccountManager) SubmitForm(str string, app *Application) types.Result {
	accounts := am.Access()
	privKeyString := util.ReadPrivKeyString(str)
	account := accounts[privKeyString]
	done := make(chan struct{}, 1)
	go am.Return(accounts, done)
	select {
	case <-done:
		if account == nil {
			return types.NewResult(types.CodeType_InternalError, nil, "account with private key does not exist")
		}
		return account.SubmitForm(util.RemovePrivKeyString(str), app)
	}
}

func (am *AccountManager) QueryForm(str string, cache *Cache) (*Form, error) {
	accounts := am.Access()
	privKeyString := util.ReadPrivKeyString(str)
	account := accounts[privKeyString]
	done := make(chan struct{}, 1)
	go am.Return(accounts, done)
	select {
	case <-done:
		if account == nil {
			return nil, errors.New("account with private key does not exist")
		}
		return account.QueryForm(util.RemovePrivKeyString(str), cache)
	}
}
