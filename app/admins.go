package app

import (
	"errors"
	. "github.com/tendermint/go-crypto"
	util "github.com/zballs/3ii/util"
	"log"
)

// Admin

type Admin struct {
	Account
}

func (ad *Admin) ResolveForm(str string, cache *Cache) error {
	pubKeyString := util.ReadPubKeyString(str)
	if !ad.Validate(pubKeyString) {
		return errors.New("invalid public-private key pair")
	}
	id := util.ReadFormID(str)
	return cache.ResolveForm(id)
}

// Admins, Admindb
type Admins map[string]*Admin
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

func (db *Admindb) AdminAccess() Admins {
	return <-(*db).channel
}

func (db *Admindb) AdminRestore(admins Admins, done chan struct{}) {
	(*db).channel <- admins
	done <- struct{}{}
}

func (db *Admindb) AdminAdd(privkey PrivKeyEd25519, admin *Admin) error {
	admins := db.AdminAccess()
	if admins[util.PrivKeyToString(privkey)] != nil {
		return errors.New("admin with private key already exists")
	} else if len(admins) == db.capacity {
		return errors.New("admin db full")
	}
	admins[util.PrivKeyToString(privkey)] = admin
	done := make(chan struct{}, 1)
	go db.AdminRestore(admins, done)
	select {
	case <-done:
		return nil
	}
}

func (db *Admindb) AdminRemove(pubKeyString string, privKeyString string) error {
	var err error = nil
	admins := db.AdminAccess()
	admin := admins[privKeyString]
	if admin != nil {
		if admin.Validate(pubKeyString) {
			delete(admins, privKeyString)
		} else {
			err = errors.New("invalid public-private key pair")
		}
	} else {
		err = errors.New("admin with private key does not exist")
	}
	done := make(chan struct{}, 1)
	go db.AdminRestore(admins, done)
	select {
	case <-done:
		return err
	}
}

// Admin Manager
type AdminManager struct {
	Admindb
	*AccountManager
}

func CreateAdminManager(db_capacity int) *AdminManager {
	return &AdminManager{CreateAdmindb(db_capacity), CreateAccountManager()}
}

func (am *AdminManager) CreateAdmin(passphrase string) (pubkey PubKeyEd25519, privkey PrivKeyEd25519, err error) {
	var account = Account{}
	var admin = Admin{}
	secret := util.GenerateSecret([]byte(passphrase))
	privkey = GenPrivKeyEd25519FromSecret(secret)
	copy(pubkey[:], privkey.PubKey().Bytes())
	account.CopyBytes(pubkey[:])
	admin.CopyBytes(pubkey[:])
	err = am.Add(privkey, &account)
	err = am.AdminAdd(privkey, &admin)
	return pubkey, privkey, err
}

func (am *AdminManager) RemoveAdmin(pubKeyString string, privKeyString string) error {
	err := am.AdminRemove(pubKeyString, privKeyString)
	if err != nil {
		return err
	}
	err = am.Remove(pubKeyString, privKeyString)
	if err != nil {
		return err
	}
	return nil
}

func (am *AdminManager) ResolveForm(str string, cache *Cache) error {
	admins := am.AdminAccess()
	privKeyString := util.ReadPrivKeyString(str)
	admin := admins[privKeyString]
	done := make(chan struct{}, 1)
	go am.AdminRestore(admins, done)
	select {
	case <-done:
		if admin == nil {
			return errors.New("admin with private key does not exist")
		}
		return admin.ResolveForm(util.RemovePrivKeyString(str), cache)
	}
}
