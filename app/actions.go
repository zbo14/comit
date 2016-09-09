package app

import (
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-crypto"
	lib "github.com/zballs/3ii/lib"
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
// Add prepends to values..

func (al ActionListener) Run(app *Application) {
	al.On("connection", func(so socketio.Socket) {
		log.Println("connected")

		// Create Accounts
		so.On("create-account", func(secret string) {
			privkey, err := app.account_manager.CreateAccount([]byte(secret))
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println(fmt.Sprintf("created account: PubKeyEd25519{%X}", privkey.PubKey().Bytes()))
			}
		})

		// Remove Accounts
		so.On("remove-account", func(privKeyString string) {
			var privkey PrivKeyEd25519
			copy(privkey[:], []byte(privKeyString)[:])
			err := app.account_manager.RemoveAccount(privkey)
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println(fmt.Sprintf("removed account: PubKeyEd25519{%X}", privkey.PubKey().Bytes()))
			}
		})

		// Submit Forms
		// Clean up!!
		so.On("submit-form", func(t string, a string, d string, sf string, pk string) {
			tx := lib.SERVICE.WriteType([]byte(t))
			tx = append(tx, lib.SERVICE.WriteAddress([]byte(a))...)
			tx = append(tx, lib.SERVICE.WriteDescription([]byte(d))...)
			tx = append(tx, lib.SERVICE.WriteSpecField([]byte(sf), []byte(t))...)
			tx = append(tx, lib.SERVICE.WritePrivkeyBytes([]byte(pk))...)
			log.Println(string(tx))
			// result := app.account_manager.SubmitForm(tx, app)
			// log.Println(result.Log)
		})

		// Query Forms
		so.On("query-form", func(str string) {
			tx := []byte(str)
			form := app.account_manager.QueryForm(tx, app.cache)
			if form != nil {
				log.Println(*form)
			} else {
				log.Println("no form found")
			}
		})

		// Query Resolved Forms
		so.On("query-resolved", func(str string) {
			tx := []byte(str)
			form := app.account_manager.QueryResolved(tx, app.cache)
			if form != nil {
				log.Println(*form)
			} else {
				log.Println("no form found")
			}
		})

		// Disconnect
		al.On("disconnection", func() {
			log.Println("disconnected")
		})
	})

	// Errors
	al.On("error", func(so socketio.Socket, err error) {
		log.Println(err.Error())
	})
}
