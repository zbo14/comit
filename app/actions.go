package app

import (
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-crypto"
	"log"
)

type ActionListener struct {
	*socketio.Server
}

func CreateActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	return ActionListener{server}, err
}

// Change print statements to socket emit statements

func (al ActionListener) CreateAccounts(am *AccountManager) {
	al.On("create-account", func(secret string) {
		privkey, err := (*am).CreateAccount([]byte(secret))
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Println(fmt.Sprintf("created account: PubKeyEd25519{%X}", privkey.PubKey().Bytes()))
		}
	})
}
func (al ActionListener) RemoveAccounts(am *AccountManager) {
	al.On("remove-account", func(privKeyString string) {
		var privkey PrivKeyEd25519
		copy(privkey[:], []byte(privKeyString)[:])
		err := (*am).RemoveAccount(privkey)
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Println(fmt.Sprintf("removed account: PubKeyEd25519{%X}", privkey.PubKey().Bytes()))
		}
	})
}

func (al ActionListener) SubmitForms(am *AccountManager, app *Application) {
	al.On("submit-form", func(str string) {
		tx := []byte(str)
		result := (*am).SubmitForm(tx, app)
		log.Println(result.Log)
	})
}

func (al ActionListener) QueryForms(am *AccountManager, cache *Cache) {
	al.On("query-form", func(str string) {
		tx := []byte(str)
		form := (*am).QueryForm(tx, cache)
		if form != nil {
			log.Println(*form)
		} else {
			log.Println("no form found")
		}
	})
}

func (al ActionListener) QueryResolved(am *AccountManager, cache *Cache) {
	al.On("query-resolved", func(str string) {
		tx := []byte(str)
		form := (*am).QueryResolved(tx, cache)
		if form != nil {
			log.Println(*form)
		} else {
			log.Println("no form found")
		}
	})
}
