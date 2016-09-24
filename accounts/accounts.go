package accounts

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-p2p"
)

// Account
func accountToPubKeyString(account *Switch) string {
	return fmt.Sprintf("%x", account.NodeInfo().PubKey[:])
}

func accountPassphrase(account *Switch) string {
	return account.NodeInfo().Other[0]
}

func validateAccount(passphrase string, account *Switch) bool {
	return passphrase == accountPassphrase(account)
}

// Accounts, Accountdb

type Accounts map[string]*Switch
type Accountdb chan Accounts

func createAccountdb() Accountdb {
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

func (db Accountdb) access() Accounts {
	return <-db
}

func (db Accountdb) restore(accounts Accounts, done chan struct{}) {
	db <- accounts
	done <- struct{}{}
}

func (db Accountdb) add(account *Switch) error {
	accounts := db.access()
	pubKeyString := accountToPubKeyString(account)
	if accounts[pubKeyString] != nil {
		return errors.New(account_already_exists)
	}
	accounts[pubKeyString] = account
	done := make(chan struct{}, 1)
	go db.restore(accounts, done)
	select {
	case <-done:
		return nil
	}
}

func (db Accountdb) remove(pubKeyString string, passphrase string) (err error) {
	accounts := db.access()
	account := accounts[pubKeyString]
	if account != nil {
		if validateAccount(passphrase, account) {
			delete(accounts, pubKeyString)
		} else {
			err = errors.New(invalid_pubkey_passphrase)
		}
	} else {
		err = errors.New(account_not_found)
	}
	done := make(chan struct{}, 1)
	go db.restore(accounts, done)
	select {
	case <-done:
		return err
	}
}

// Account Manager

type AccountManager struct {
	Accountdb
}

func createAccountManager() *AccountManager {
	return &AccountManager{createAccountdb()}
}

func (acm *AccountManager) Authorize(pubKeyString string, passphrase string) error {
	accounts := acm.access()
	account := accounts[pubKeyString]
	done := make(chan struct{}, 1)
	go acm.restore(accounts, done)
	select {
	case <-done:
		if account == nil {
			return errors.New(account_not_found)
		}
		if !validateAccount(passphrase, account) {
			return errors.New(invalid_pubkey_passphrase)
		}
		return nil
	}
}

func (acm *AccountManager) Remove(pubKeyString string, passphrase string) error {
	return acm.remove(pubKeyString, passphrase)
}

func (acm *AccountManager) Broadcast(pubKeyString string, str string, chIDs ...uint8) {
	accounts := acm.access()
	account := accounts[pubKeyString]
	done := make(chan struct{}, 1)
	go acm.restore(accounts, done)
	select {
	case <-done:
		if account.IsRunning() {
			for _, chID := range chIDs {
				account.Broadcast(chID, str)
			}
		}
	}
}
