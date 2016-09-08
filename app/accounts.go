package app

import (
	"errors"
	types "github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/util"
)

// Account

type Account struct {
	pubkey  Pubkey
	privkey Privkey
}

func (ac *Account) SubmitRequest(app *Application, tx []byte) types.Result {
	signature := append([]byte("pubkey:"), (*ac).pubkey[:]...)
	tx = append(tx, signature...)
	return app.AppendTx(tx)
}

func (*Account) QueryForm(cache *Cache, tx []byte) *Form {
	return (*cache).QueryForm(string(tx))
}

func (*Account) QueryResolved(cache *Cache, tx []byte) *Form {
	return (*cache).QueryResolved(string(tx))
}

// Account Manager

type AccountManager struct {
	Accounts chan map[string]*Account
}

func CreateAccountManager() *AccountManager {
	return &AccountManager{make(chan map[string]*Account, 1)}
}

func (am *AccountManager) CreateAccount(tx []byte) Privkey {
	account := Account{
		pubkey:  GeneratePubkey(tx),
		privkey: GeneratePrivkey(tx),
	}
	accounts := <-am.Accounts
	accounts[string(account.privkey[:])] = &account
	done := make(chan struct{}, 1)
	go func() {
		am.Accounts <- accounts
		done <- struct{}{}
	}()
	select {
	case <-done:
		return account.privkey
	}
}

func (am *AccountManager) RemoveAccount(privkey Privkey) error {
	accounts := <-am.Accounts
	account := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go func() {
		am.Accounts <- accounts
		done <- struct{}{}
	}()
	select {
	case <-done:
		if account == nil {
			return errors.New("account with private key does not exist")
		}
		return nil
	}
}

func (am *AccountManager) SubmitRequest(privkey Privkey, app *Application, tx []byte) types.Result {
	accounts := <-am.Accounts
	account := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go func() {
		am.Accounts <- accounts
		done <- struct{}{}
	}()
	select {
	case <-done:
		if account == nil {
			return types.NewResult(types.CodeType_InternalError, nil, "account with private key does not exist")
		}
		return (*account).SubmitRequest(app, tx)
	}
}

func (am *AccountManager) QueryForm(privkey Privkey, cache *Cache, tx []byte) *Form {
	accounts := <-am.Accounts
	account := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go func() {
		am.Accounts <- accounts
		done <- struct{}{}
	}()
	select {
	case <-done:
		if account == nil {
			return nil
		}
		return (*account).QueryForm(cache, tx)
	}
}

func (am *AccountManager) QueryResolved(privkey Privkey, cache *Cache, tx []byte) *Form {
	accounts := <-am.Accounts
	account := accounts[string(privkey[:])]
	done := make(chan struct{}, 1)
	go func() {
		am.Accounts <- accounts
		done <- struct{}{}
	}()
	select {
	case <-done:
		if account == nil {
			return nil
		}
		return (*account).QueryResolved(cache, tx)
	}
}
