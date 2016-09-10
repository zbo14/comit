package app

import (
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	// . "github.com/tendermint/go-crypto"
	lib "github.com/zballs/3ii/lib"
	util "github.com/zballs/3ii/util"
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

func (al ActionListener) Run(app *Application) {
	al.On("connection", func(so socketio.Socket) {
		log.Println("connected")

		// Create Accounts
		so.On("create-account", func(passphrase string) {
			pubkey, privkey, err := app.account_manager.CreateAccount(passphrase)
			if err != nil {
				log.Println(err.Error())
			}
			so.Emit("return-keys", util.PubKeyToString(pubkey), util.PrivKeyToString(privkey))
		})

		// Remove Accounts
		so.On("remove-account", func(pubKeyString string, privKeyString string) {
			err := app.account_manager.RemoveAccount(pubKeyString, privKeyString)
			if err != nil {
				log.Println(err.Error())
			} else {
				so.Emit("remove-msg", fmt.Sprintf("remove account [PubKeyEd25519{%v}, PrivKeyEd25519{%v}]", pubKeyString, privKeyString))
			}
		})

		// Submit Forms
		so.On("submit-form", func(_type string, _address string, _description string, _specfield string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteType(_type) + lib.SERVICE.WriteAddress(_address) + lib.SERVICE.WriteDescription(_description) + lib.SERVICE.WriteSpecField(_specfield, _type) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			result := app.account_manager.SubmitForm(str, app)
			so.Emit("return-formID", result.Log)
		})

		// Query Forms
		so.On("query-form", func(_formID string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteFormID(_formID) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			form, err := app.account_manager.QueryForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
			} else {
				so.Emit("return-form", ParseForm(form))
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(_formID string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteFormID(_formID) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			err := app.account_manager.ResolveForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
			} else {
				so.Emit("resolve-msg", "resolved form with ID "+_formID)
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
