package app

import (
	"errors"
	. "github.com/tendermint/go-p2p"
	util "github.com/zballs/3ii/util"
)

// Admins, Admindb
type Admins map[string]*Switch
type Admindb struct {
	channel  chan Admins
	capacity int
}

func getDept(admin *Switch) (dept string) {
	return admin.NodeInfo().Other[1]
}

func getServices(admin *Switch) (services []string) {
	return admin.NodeInfo().Other[2:]
}

func registerUserAsAdmin(user *Switch, dept string, services []string, sendr *Switch) {
	user.NodeInfo().Other = append(user.NodeInfo().Other, dept)
	user.NodeInfo().Other = append(user.NodeInfo().Other, services...)
	AddReactor(user, AdminChannelIDs, "admin")
	Connect2Switches(sendr, user)
}

func createAdmindb(capacity int) Admindb {
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

func (db Admindb) accessAdmins() Admins {
	return <-db.channel
}

func (db Admindb) restoreAdmins(admins Admins, done chan struct{}) {
	db.channel <- admins
	done <- struct{}{}
}

func (db Admindb) addAdmin(admin *Switch) error {
	admins := db.accessAdmins()
	pubKeyString := userToPubKeyString(admin)
	if admins[pubKeyString] != nil {
		return errors.New(admin_already_exists)
	} else if len(admins) == db.capacity {
		return errors.New(admin_db_full)
	}
	admins[pubKeyString] = admin
	done := make(chan struct{}, 1)
	go db.restoreAdmins(admins, done)
	select {
	case <-done:
		return nil
	}
}

func (db Admindb) removeAdmin(pubKeyString string, passphrase string) (err error) {
	admins := db.accessAdmins()
	admin := admins[pubKeyString]
	if admin != nil {
		if validateUser(passphrase, admin) {
			delete(admins, pubKeyString)
		} else {
			err = errors.New(invalid_pubkey_passphrase)
		}
	} else {
		err = errors.New(admin_not_found)
	}
	done := make(chan struct{}, 1)
	go db.restoreAdmins(admins, done)
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
		createAdmindb(db_capacity),
		CreateUserManager(),
	}
}

func (am *AdminManager) RegisterAdmin(dept string, services []string, passphrase string, recvr *Switch, sendr *Switch) (pubKeyString string, privKeyString string, err error) {
	pubKeyString, privKeyString, err = am.RegisterUser(passphrase, recvr)
	if err != nil {
		return
	}
	users := am.accessUsers()
	user := users[pubKeyString]
	done := make(chan struct{}, 1)
	go am.restoreUsers(users, done)
	select {
	case <-done:
		if user == nil {
			err = errors.New(user_not_found)
			return
		}
		registerUserAsAdmin(user, dept, services, sendr)
		err = am.addAdmin(user)
		return
	}
}

func (am *AdminManager) RemoveAdmin(pubKeyString string, passphrase string) error {
	err := am.RemoveUser(pubKeyString, passphrase)
	if err != nil {
		return err
	}
	err = am.removeAdmin(pubKeyString, passphrase)
	if err != nil {
		return err
	}
	return nil
}

func (am *AdminManager) ResolveForm(str string, cache *Cache) error {
	admins := am.accessAdmins()
	pubKeyString := util.ReadPubKeyString(str)
	admin := admins[pubKeyString]
	done := make(chan struct{}, 1)
	go am.restoreAdmins(admins, done)
	select {
	case <-done:
		if admin == nil {
			return errors.New(admin_not_found)
		}
		passphrase := util.ReadPassphrase(str)
		if !validateUser(passphrase, admin) {
			return errors.New(invalid_pubkey_passphrase)
		}
		formID := util.ReadFormID(util.RemovePassphrase(str))
		return cache.ResolveForm(formID)
	}
}

func (am *AdminManager) FindAdmin(pubKeyString string, passphrase string) (*Switch, []string, error) {
	admins := am.accessAdmins()
	admin := admins[pubKeyString]
	done := make(chan struct{}, 1)
	go am.restoreAdmins(admins, done)
	select {
	case <-done:
		if admin == nil {
			return nil, nil, errors.New(admin_not_found)
		} else if !validateUser(passphrase, admin) {
			return nil, nil, errors.New(invalid_pubkey_passphrase)
		}
		return admin, getServices(admin), nil
	}
}
