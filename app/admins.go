package app

import (
	"errors"
	. "github.com/tendermint/go-p2p"
	util "github.com/zballs/3ii/util"
)

// Admin
// func RegisterAdmin()

// Admins, Admindb
type Admins map[string]*Switch
type Admindb struct {
	channel  chan Admins
	capacity int
}

func CreateAdmindb(capacity int) Admindb {
	admindb := Admindb{make(chan Admins, 1), capacity}
	done := make(chan struct{}, 1)
	go func() {
		admindb.channel <- Admins{}
		done <- struct{}{}
	}()
	select {
	case <-done:
		return admindb
	}
}

func (db Admindb) AccessAdmins() Admins {
	return <-db.channel
}

func (db Admindb) RestoreAdmins(admins Admins, done chan struct{}) {
	db.channel <- admins
	done <- struct{}{}
}

func (db Admindb) AddAdmin(admin *Switch) error {
	admins := db.AccessAdmins()
	pubKeyString := UserToPubKeyString(admin)
	if admins[pubKeyString] != nil {
		return errors.New("admin with public key already exists")
	} else if len(admins) == db.capacity {
		return errors.New("admin db full")
	}
	admins[pubKeyString] = admin
	done := make(chan struct{}, 1)
	go db.RestoreAdmins(admins, done)
	select {
	case <-done:
		return nil
	}
}

func (db Admindb) RemoveAdmin(pubKeyString string, passphrase string) (err error) {
	admins := db.AccessAdmins()
	admin := admins[pubKeyString]
	if admin != nil {
		if ValidateUser(passphrase, admin) {
			delete(admins, pubKeyString)
		} else {
			err = errors.New("invalid public key + passphrase")
		}
	} else {
		err = errors.New("admin with public key does not exist")
	}
	done := make(chan struct{}, 1)
	go db.RestoreAdmins(admins, done)
	select {
	case <-done:
		return err
	}
}

// Admin Manager
type AdminManager struct {
	Admindb
	*UserManager
}

func CreateAdminManager(db_capacity int) *AdminManager {
	return &AdminManager{
		CreateAdmindb(db_capacity),
		CreateUserManager(),
	}
}

func (am *AdminManager) RegisterAdmin(passphrase string, recvr *Switch) (pubKeyString string, privKeyString string, err error) {
	pubKeyString, privKeyString, err = am.RegisterUser(passphrase, recvr)
	if err != nil {
		return
	}
	users := am.AccessUsers()
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go am.RestoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			err = errors.New("user with public key not found")
			return
		}
		err = am.AddAdmin(user)
		return
	}
}

func (am *AdminManager) RemoveAdmin(pubKeyString string, passphrase string) error {
	err := am.RemoveUser(pubKeyString, passphrase)
	if err != nil {
		return err
	}
	err = am.RemoveAdmin(pubKeyString, passphrase)
	if err != nil {
		return err
	}
	return nil
}

func (am *AdminManager) ResolveForm(str string, cache *Cache) error {
	admins := am.AccessAdmins()
	pubKeyString := util.ReadPubKeyString(str)
	admin := admins[pubKeyString]
	done := make(chan struct{}, 1)
	go am.RestoreAdmins(admins, done)
	select {
	case <-done:
		if admin == nil {
			return errors.New("admin with private key not found")
		}
		passphrase := util.ReadPassphrase(str)
		if !ValidateUser(passphrase, admin) {
			return errors.New("invalid public key + passphrase")
		}
		formID := util.ReadFormID(util.RemovePassphrase(str))
		return cache.ResolveForm(formID)
	}
}
