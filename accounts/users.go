package accounts

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	. "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
)

// User

func userToPubKeyString(user *Switch) string {
	return fmt.Sprintf("%x", user.NodeInfo().PubKey[:])
}

func registerUser(passphrase string, recvr *Switch) (user *Switch, pubKeyString string, privKeyString string) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	user = CreateSwitch(privkey, passphrase)
	AddReactor(user, FeedChannelIDs, "feed")
	Connect2Switches(user, recvr)
	pubKeyString = userToPubKeyString(user)
	privKeyString = util.PrivKeyToString(privkey)
	return
}

func validateUser(passphrase string, user *Switch) bool {
	return passphrase == user.NodeInfo().Other[0]
}

// Users, Userdb

type Users map[string]*Switch
type Userdb chan Users

func createUserdb() Userdb {
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

func (db Userdb) accessUsers() Users {
	return <-db
}

func (db Userdb) restoreUsers(users Users, done chan struct{}) {
	db <- users
	done <- struct{}{}
}

func (db Userdb) addUser(user *Switch) error {
	users := db.accessUsers()
	pubKeyString := userToPubKeyString(user)
	if users[pubKeyString] != nil {
		return errors.New(user_already_exists)
	}
	users[pubKeyString] = user
	done := make(chan struct{}, 1)
	go db.restoreUsers(users, done)
	select {
	case <-done:
		return nil
	}
}

func (db Userdb) removeUser(pubKeyString string, passphrase string) (err error) {
	users := db.accessUsers()
	user := users[pubKeyString]
	if user != nil {
		if validateUser(passphrase, user) {
			delete(users, pubKeyString)
		} else {
			err = errors.New(invalid_pubkey_passphrase)
		}
	} else {
		err = errors.New(user_not_found)
	}
	done := make(chan struct{}, 1)
	go db.restoreUsers(users, done)
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
	return &UserManager{createUserdb()}
}

func (um *UserManager) RegisterUser(passphrase string, recvr *Switch) (string, string, error) {
	user, pubKeyString, privKeyString := registerUser(passphrase, recvr)
	err := um.addUser(user)
	return pubKeyString, privKeyString, err
}

func (um *UserManager) RemoveUser(pubKeyString string, passphrase string) error {
	return um.removeUser(pubKeyString, passphrase)
}

func (um *UserManager) AuthorizeUser(pubKeyString string, passphrase string) error {
	users := um.accessUsers()
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.restoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			return errors.New(user_not_found)
		}
		if !validateUser(passphrase, user) {
			return errors.New(invalid_pubkey_passphrase)
		}
		return nil
	}
}

func (um *UserManager) UserIsRunning(pubKeyString string) bool {
	users := um.accessUsers()
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.restoreUsers(users, done)
	select {
	case <-done:
		return user.IsRunning()
	}
}

func (um *UserManager) UserBroadcast(pubKeyString string, str string, chID uint8) {
	users := um.accessUsers()
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go um.restoreUsers(users, done)
	select {
	case <-done:
		user.Broadcast(FeedChannelIDs["general"], str)
		if chID > uint8(0) {
			user.Broadcast(chID, str)
		}
	}
}
