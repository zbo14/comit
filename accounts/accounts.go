package accounts

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	. "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
)

type Account struct {
	*Switch
	passphrase string
}

func registerAccount(passphrase string) (*Account, string, string, error) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	account := &Account{Switch: CreateSwitch(privkey)}
	account.setPassphrase(passphrase, privkey)
	AddReactor(account.Switch, DeptChannelIDs(), "dept-feed")
	AddReactor(account.Switch, ServiceChannelIDs(), "service-feed")
	account.Start()
	_, err := DialPeerWithAddr(account.Switch, RecvrListenerAddr())
	if err != nil {
		return nil, "", "", err
	}
	pubKeyString := account.toPubKeyString()
	privKeyString := util.PrivKeyToString(privkey)
	return account, pubKeyString, privKeyString, nil
}

// Account
func (account *Account) toPubKeyString() string {
	return fmt.Sprintf("%x", account.NodeInfo().PubKey[:])
}

func (account *Account) getPassphrase() string {
	return account.passphrase
}

func (account *Account) setPassphrase(passphrase string, privkey PrivKeyEd25519) {
	if account.toPubKeyString() == util.PubKeyToString(privkey.PubKey().(PubKeyEd25519)) {
		account.passphrase = passphrase
	}
}

func (account *Account) validateAccount(passphrase string) bool {
	return passphrase == account.getPassphrase()
}

// Accounts, Accountdb

type Accounts map[string]*Account
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

func (db Accountdb) add(account *Account) error {
	accounts := db.access()
	pubKeyString := account.toPubKeyString()
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
		if account.validateAccount(passphrase) {
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
		if !account.validateAccount(passphrase) {
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
