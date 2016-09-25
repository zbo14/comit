package accounts

import (
	"errors"
)

// User Manager

type UserManager struct {
	*AccountManager
}

func CreateUserManager() *UserManager {
	return &UserManager{createAccountManager()}
}

func (um *UserManager) Register(passphrase string) (string, string, error) {
	user, pubKeyString, privKeyString, err := registerAccount(passphrase)
	if err != nil {
		return "", "", errors.New(user_network_fail)
	}
	err = um.add(user)
	return pubKeyString, privKeyString, err
}
