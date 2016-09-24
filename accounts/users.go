package accounts

import (
	"errors"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	. "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
)

// Register User

func registerUser(passphrase string) (user *Switch, pubKeyString string, privKeyString string, err error) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	user = CreateSwitch(privkey, passphrase)
	AddReactor(user, DeptChannelIDs(), "dept-feed")
	AddReactor(user, ServiceChannelIDs(), "service-feed")
	user.Start()
	_, err = DialPeerWithAddr(user, RecvrListenerAddr())
	if err != nil {
		return
	}
	pubKeyString = accountToPubKeyString(user)
	privKeyString = util.PrivKeyToString(privkey)
	return
}

// User Manager

type UserManager struct {
	*AccountManager
}

func CreateUserManager() *UserManager {
	return &UserManager{createAccountManager()}
}

func (um *UserManager) Register(passphrase string) (string, string, error) {
	user, pubKeyString, privKeyString, err := registerUser(passphrase)
	if err != nil {
		return "", "", errors.New(user_network_fail)
	}
	err = um.add(user)
	return pubKeyString, privKeyString, err
}
