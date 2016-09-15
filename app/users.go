package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	types "github.com/tendermint/tmsp/types"
	lib "github.com/zballs/3ii/lib"
	util "github.com/zballs/3ii/util"
	"log"
)

// User Switch

func UserToPubKeyString(user *Switch) string {
	return fmt.Sprintf("%x", user.NodeInfo().PubKey[:])
}

func RegisterUser(passphrase string, recvr *Switch) (user *Switch, pubKeyString string, privKeyString string) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	user = StartSwitch(privkey, passphrase)
	Connect2Switches(user, recvr)
	pubKeyString = UserToPubKeyString(user)
	privKeyString = util.PrivKeyToString(privkey)
	log.Println(user.Peers().Size())
	log.Println(recvr.Peers().Size())
	return
}

func ValidateUser(passphrase string, user *Switch) bool {
	return passphrase == user.NodeInfo().Other[0]
}

// Users, Userdb

type Users map[string]*Switch
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

func (db Userdb) AddUser(user *Switch) error {
	users := db.AccessUsers()
	pubKeyString := UserToPubKeyString(user)
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
		if ValidateUser(passphrase, user) {
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
	recvr *Switch
}

func CreateUserManager() *UserManager {
	return &UserManager{
		CreateUserdb(),
		StartSwitch(GenPrivKeyEd25519(), ""),
	}
}

func (um *UserManager) RegisterUser(passphrase string) (string, string, error) {
	user, pubKeyString, privKeyString := RegisterUser(passphrase, um.recvr)
	err := um.AddUser(user)
	return pubKeyString, privKeyString, err
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
		if !ValidateUser(passphrase, user) {
			return types.NewResult(
				types.CodeType_Unauthorized,
				nil,
				"invalid public key + passphrase",
			)
		}
		txstr := util.RemovePassphrase(str)
		tx := []byte(txstr)
		result := app.AppendTx(tx)
		if result.IsOK() && user.IsRunning() {
			user.Broadcast(byte(0x00), txstr)
		}
		return result
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
		if !ValidateUser(passphrase, user) {
			return nil, errors.New("invalid public key + passphrase")
		}
		formID := util.ReadFormID(util.RemovePassphrase(str))
		return cache.FindForm(formID)
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
		if !ValidateUser(passphrase, user) {
			return nil, errors.New("invalid public key + passphrase")
		}
		return cache.SearchForms(util.RemovePassphrase(str), _status), nil
	}
}

func FormatUpdate(update PeerMessage) string {
	str := string(update.Bytes)
	_type := lib.SERVICE.ReadField(str, "type")
	_address := lib.SERVICE.ReadField(str, "address")
	_description := lib.SERVICE.ReadField(str, "description")
	_specfield := lib.SERVICE.FieldOpts(_type).Field
	return "<strong>issue</strong> " + _type + "<br>" + "<strong>address</strong> " + _address + "<br>" + "<strong>description</strong> " + _description + "<br>" + fmt.Sprintf("<strong>%v</strong>", _specfield)
}

func (um *UserManager) RecvFeedUpdates(feedUpdates chan string) {
	for {
		if um.recvr.IsRunning() {
			updates := um.recvr.Reactor("feed").(*MyReactor).getMsgs(byte(0x00))
			if len(updates) > 0 {
				update := updates[len(updates)-1]
				feedUpdates <- FormatUpdate(update)
			}
		}
	}
}
