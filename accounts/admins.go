package accounts

import (
	"errors"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	. "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
)

// Admins, Admindb
func getDept(admin *Switch) (dept string) {
	return admin.NodeInfo().Other[1]
}

func getServices(admin *Switch) (services []string) {
	return admin.NodeInfo().Other[2:]
}

func registerAdmin(dept string, services []string, passphrase string) (admin *Switch, pubKeystring string, privKeyString string, err error) {
	secret := util.GenerateSecret([]byte(passphrase))
	privkey := GenPrivKeyEd25519FromSecret(secret)
	admin = CreateSwitch(privkey, passphrase)
	admin.NodeInfo().Other = append(admin.NodeInfo().Other, dept)
	admin.NodeInfo().Other = append(admin.NodeInfo().Other, services...)
	AddReactor(admin, DeptChannelIDs(), "dept-feed")
	AddReactor(admin, ServiceChannelIDs(), "service-feed")
	admin.Start()
	_, err = DialPeerWithAddr(admin, RecvrListenerAddr())
	if err != nil {
		return
	}
	pubKeystring = accountToPubKeyString(admin)
	privKeyString = util.PrivKeyToString(privkey)
	return
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

func (adm *AdminManager) Register(dept string, services []string, passphrase string) (string, string, error) {
	admins := adm.access()
	admin_count := len(admins)
	done := make(chan struct{}, 1)
	go adm.restore(admins, done)
	select {
	case <-done:
		if !(admin_count < adm.db_capacity) {
			return "", "", errors.New(admin_db_full)
		}
		admin, pubKeyString, privKeyString, err := registerAdmin(dept, services, passphrase)
		if err != nil {
			return "", "", errors.New(admin_network_fail)
		}
		err = adm.add(admin)
		return pubKeyString, privKeyString, err
	}
}
