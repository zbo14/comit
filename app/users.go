package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-common"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	types "github.com/tendermint/tmsp/types"
	util "github.com/zballs/3ii/util"
)

// User
type User struct {
	*Peer
}

func CreateUser(pubkey PubKeyEd25519) *User {
	var u = &User{&Peer{}}
	u.Key = util.PubKeyToString(pubkey)
	u.Data = NewCMap()
	return u
}

func (u *User) Validate(pubKeyString string) bool {
	return pubKeyString == u.Key
}

func (u *User) SubmitForm(str string, app *Application) types.Result {
	pubKeyString := util.ReadPubKeyString(str)
	if !u.Validate(pubKeyString) {
		return types.NewResult(types.CodeType_InternalError, nil, "invalid public-private key pair")
	}
	tx := []byte(str)
	return app.AppendTx(tx)
}

func (u *User) FindForm(str string, cache *Cache) (*Form, error) {
	pubKeyString := util.ReadPubKeyString(str)
	if !u.Validate(pubKeyString) {
		return nil, errors.New("invalid public-private key pair")
	}
	id := util.ReadFormID(str)
	return cache.FindForm(id)
}

func (u *User) SearchForms(str string, _status string, cache *Cache) (Formlist, error) {
	pubKeyString := util.ReadPubKeyString(str)
	if !u.Validate(pubKeyString) {
		return nil, errors.New("invalid public-private key pair")
	}
	return cache.SearchForms(str, _status), nil
}

func (u *User) SendMessage(message string, pubKeyString string) error {
	if !u.Validate(pubKeyString) {
		return errors.New("invalid public-private key pair")
	}
	if !u.TrySend(byte(0), []byte(message)) {
		return errors.New("message failed to send")
	}
	return nil
}

// Users, Userdb
type Users map[string]*User
type Userdb chan Users

func CreateUserdb() Userdb {
	userdb := make(chan Users, 1)
	done := make(chan struct{}, 1)
	go func() {
		userdb <- Users{}
		done <- struct{}{}
	}()
	select {
	case <-done:
		return userdb
	}
}

func (db Userdb) dbAccess() Users {
	return <-db
}

func (db Userdb) dbRestore(users Users, done chan struct{}) {
	db <- users
	done <- struct{}{}
}

func (db Userdb) dbAdd(privkey PrivKeyEd25519, user *User) error {
	users := db.dbAccess()
	if users[util.PrivKeyToString(privkey)] != nil {
		return errors.New("user with private key already exists")
	}
	users[util.PrivKeyToString(privkey)] = user
	done := make(chan struct{}, 1)
	go db.dbRestore(users, done)
	select {
	case <-done:
		return nil
	}
}

func (db Userdb) dbRemove(pubKeyString string, privKeyString string) error {
	var err error = nil
	users := db.dbAccess()
	user := users[privKeyString]
	if user != nil {
		if user.Validate(pubKeyString) {
			delete(users, privKeyString)
		} else {
			err = errors.New("invalid public-private key pair")
		}
	} else {
		err = errors.New("user with private key does not exist")
	}
	done := make(chan struct{}, 1)
	go db.dbRestore(users, done)
	select {
	case <-done:
		return err
	}
}

type UserManager struct {
	Userdb
	*Switch
}

func CreateUserManager() *UserManager {
	um := &UserManager{
		CreateUserdb(),
		CreateSwitch(config, msgBoard),
	}
	_, err := um.Start()
	if err != nil {
		return nil
	}
	_, err = um.Reactor("msgBoard").Start()
	if err != nil {
		return nil
	}
	return um
}

func (um *UserManager) CreateUser(passphrase string) (pubkey PubKeyEd25519, privkey PrivKeyEd25519, err error) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey = GenPrivKeyEd25519FromSecret(secret)
	copy(pubkey[:], privkey.PubKey().Bytes())
	user := CreateUser(pubkey)
	um.Reactor("msgBoard").AddPeer(user.Peer)
	err = um.dbAdd(privkey, user)
	return pubkey, privkey, err
}

func (um *UserManager) RemoveUser(pubKeyString string, privKeyString string) error {
	users := um.dbAccess()
	user := users[privKeyString]
	done := make(chan struct{}, 1)
	go um.dbRestore(users, done)
	select {
	case <-done:
		um.Reactor("msgBoard").RemovePeer(user.Peer, "")
		return um.dbRemove(pubKeyString, privKeyString)
	}
}

func (um *UserManager) SubmitForm(str string, app *Application) types.Result {
	users := um.dbAccess()
	privKeyString := util.ReadPrivKeyString(str)
	user := users[privKeyString]
	done := make(chan struct{}, 1)
	go um.dbRestore(users, done)
	select {
	case <-done:
		if user == nil {
			return types.NewResult(types.CodeType_InternalError, nil, "user with private key does not exist")
		}
		return user.SubmitForm(util.RemovePrivKeyString(str), app)
	}
}

func (um *UserManager) FindForm(str string, cache *Cache) (*Form, error) {
	users := um.dbAccess()
	privKeyString := util.ReadPrivKeyString(str)
	user := users[privKeyString]
	done := make(chan struct{}, 1)
	go um.dbRestore(users, done)
	select {
	case <-done:
		if user == nil {
			return nil, errors.New("user with private key does not exist")
		}
		return user.FindForm(util.RemovePrivKeyString(str), cache)
	}
}

func (um *UserManager) SearchForms(str string, _status string, cache *Cache) (Formlist, error) {
	users := um.dbAccess()
	privKeyString := util.ReadPrivKeyString(str)
	user := users[privKeyString]
	done := make(chan struct{}, 1)
	go um.dbRestore(users, done)
	select {
	case <-done:
		if user == nil {
			return nil, errors.New("user with private key does not exist")
		}
		return user.SearchForms(util.RemovePrivKeyString(str), _status, cache)
	}
}

func (um *UserManager) SendMessage(message string, sendTo string, pubKeyString string, privKeyString string) error {
	users := um.dbAccess()
	user := users[privKeyString]
	done := make(chan struct{}, 1)
	go um.dbRestore(users, done)
	select {
	case <-done:
		if user == nil {
			return errors.New("user with private key does not exist")
		}
		if sendTo != "msgBoard" {
			return errors.New("incorrect sendTo")
		}
		return user.SendMessage(message, pubKeyString)
	}
}
