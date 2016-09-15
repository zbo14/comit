package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	types "github.com/tendermint/tmsp/types"
	util "github.com/zballs/3ii/util"
)

// User

type User struct {
	*Switch
	passphrase string
}

func CreateUser(passphrase string) (*User, string) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	return &User{StartSwitch(privkey), passphrase}, fmt.Sprintf("%x", privkey[:])
}

func (u *User) PubKeyString() string {
	return fmt.Sprintf("%x", u.NodeInfo().PubKey[:])
}

func (u *User) Register(recvr *Switch) string {
	Connect2Switches(u.Switch, recvr)
	return u.PubKeyString()
}

func (u *User) Validate(passphrase string) bool {
	return u.passphrase == passphrase
}

func (u *User) SubmitForm(str string, app *Application) types.Result {
	tx := []byte(str)
	res := app.AppendTx(tx)
	if res.IsOK() && u.IsRunning() {
		u.Broadcast(byte(0x00), str)
	}
	return res
}

func (u *User) FindForm(str string, cache *Cache) (*Form, error) {
	id := util.ReadFormID(str)
	return cache.FindForm(id)
}

func (u *User) SearchForms(str string, _status string, cache *Cache) (Formlist, error) {
	return cache.SearchForms(str, _status), nil
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

func (db Userdb) AccessUsers() Users {
	return <-db
}

func (db Userdb) RestoreUsers(users Users, done chan struct{}) {
	db <- users
	done <- struct{}{}
}

func (db Userdb) AddUser(user *User) error {
	users := db.AccessUsers()
	pubKeyString := user.PubKeyString()
	if users[pubKeyString] != nil {
		return errors.New("user with public key already exists")
	}
	users[pubKeyString] = user
	done := make(chan struct{}, 1)
	go db.RestoreUsers(users, done)
	select {
	case <-done:
		return nil
	}
}

func (db Userdb) RemoveUser(pubKeyString string, passphrase string) (err error) {
	users := db.AccessUsers()
	user := users[pubKeyString]
	if user != nil {
		if user.Validate(passphrase) {
			delete(users, pubKeyString)
		} else {
			err = errors.New("invalid public key + passphrase")
		}
	} else {
		err = errors.New("user with public key not found")
	}
	done := make(chan struct{}, 1)
	go db.RestoreUsers(users, done)
	select {
	case <-done:
		return err
	}
}

// User Manager

type UserManager struct {
	Userdb
}

func CreateUserManager() *UserManager {
	return &UserManager{CreateUserdb()}
}

func (um *UserManager) RegisterUser(passphrase string, recvr *Switch) (pubKeyString string, privKeyString string, err error) {
	user, privKeyString := CreateUser(passphrase)
	pubKeyString = user.Register(recvr)
	err = um.AddUser(user)
	return
}

func (um *UserManager) RemoveUser(pubKeyString string, passphrase string) error {
	return um.RemoveUser(pubKeyString, passphrase)
}

func (um *UserManager) SubmitForm(str string, app *Application) types.Result {
	users := um.AccessUsers()
	pubKeyString := util.ReadPubKeyString(str)
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.RestoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			return types.NewResult(
				types.CodeType_InternalError,
				nil,
				"user with public key not found",
			)
		}
		passphrase := util.ReadPassphrase(str)
		if !user.Validate(passphrase) {
			return types.NewResult(
				types.CodeType_Unauthorized,
				nil,
				"invalid public key + passphrase",
			)
		}
		return user.SubmitForm(
			util.RemovePassphrase(str),
			app,
		)
	}
}

func (um *UserManager) FindForm(str string, cache *Cache) (*Form, error) {
	users := um.AccessUsers()
	pubKeyString := util.ReadPubKeyString(str)
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.RestoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			return nil, errors.New("user with public key not found")
		}
		passphrase := util.ReadPassphrase(str)
		if !user.Validate(passphrase) {
			return nil, errors.New("invalid public key + passphrase")
		}
		return user.FindForm(
			util.RemovePassphrase(str),
			cache,
		)
	}
}

func (um *UserManager) SearchForms(str string, _status string, cache *Cache) (Formlist, error) {
	users := um.AccessUsers()
	pubKeyString := util.ReadPubKeyString(str)
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.RestoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			return nil, errors.New("user with public key not found")
		}
		passphrase := util.ReadPassphrase(str)
		if !user.Validate(passphrase) {
			return nil, errors.New("invalid public key + passphrase")
		}
		return user.SearchForms(
			util.RemovePassphrase(str),
			_status,
			cache,
		)
	}
}
