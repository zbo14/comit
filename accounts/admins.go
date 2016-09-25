package accounts

import (
	"errors"
)

// Admins, Admindb
func (admin *Account) getInfo() []string {
	return admin.NodeInfo().Other
}

func (admin *Account) setInfo(dept string, position string, passphrase string) {
	if admin.validateAccount(passphrase) {
		admin.NodeInfo().Other = []string{dept, position}
	}
}

func registerAdmin(dept string, position string, passphrase string) (*Account, string, string, error) {
	admin, pubKeyString, privKeyString, err := registerAccount(passphrase)
	if err != nil {
		return nil, "", "", err
	}
	admin.setInfo(dept, position, passphrase)
	return admin, pubKeyString, privKeyString, nil
}

// Admin Manager
type AdminManager struct {
	*AccountManager
	db_capacity int
}

func CreateAdminManager(db_capacity int) *AdminManager {
	return &AdminManager{
		createAccountManager(),
		db_capacity,
	}
}

func (adm *AdminManager) Register(dept string, position string, passphrase string) (string, string, error) {
	admins := adm.access()
	admin_count := len(admins)
	done := make(chan struct{}, 1)
	go adm.restore(admins, done)
	select {
	case <-done:
		if !(admin_count < adm.db_capacity) {
			return "", "", errors.New(admin_db_full)
		}
		admin, pubKeyString, privKeyString, err := registerAdmin(dept, position, passphrase)
		if err != nil {
			return "", "", errors.New(admin_network_fail)
		}
		err = adm.add(admin)
		return pubKeyString, privKeyString, err
	}
}
