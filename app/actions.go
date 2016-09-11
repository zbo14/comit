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
			pubkey, privkey, err := app.admin_manager.CreateAccount(passphrase)
			if err != nil {
				log.Println(err.Error())
			}
			msg := fmt.Sprintf("Your public-key is %v<br>Your private-key is %v<br>Do not lose it or give it to anyone!", util.PubKeyToString(pubkey), util.PrivKeyToString(privkey))
			so.Emit("keys-msg", msg)
		})

		// Create Admins
		so.On("create-admin", func(passphrase string) {
			pubkey, privkey, err := app.admin_manager.CreateAdmin(passphrase)
			if err != nil {
				log.Println(err.Error())
				msg := fmt.Sprintf("You are not authorized to create admin account")
				so.Emit("admin-keys-msg", msg)
			} else {
				msg := fmt.Sprintf("Your public-key is %v<br>Your private-key is %v<br>Do not lose it or give it to anyone!", util.PubKeyToString(pubkey), util.PrivKeyToString(privkey))
				so.Emit("admin-keys-msg", msg)
			}
		})

		// Remove Accounts
		so.On("remove-account", func(pubKeyString string, privKeyString string) {
			err := app.admin_manager.RemoveAccount(pubKeyString, privKeyString)
			if err != nil {
				log.Println(err.Error())
				so.Emit("remove-msg", fmt.Sprintf("could not remove account [public-key{%v}, private-key{%v}]", pubKeyString, privKeyString))
			} else {
				so.Emit("remove-msg", fmt.Sprintf("removed account [public-key{%v}, private-key{%v}]", pubKeyString, privKeyString))
			}
		})

		// Remove Admins
		so.On("remove-admin", func(pubKeyString string, privKeyString string) {
			err := app.admin_manager.RemoveAdmin(pubKeyString, privKeyString)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-remove-msg", fmt.Sprintf("could not remove admin [public-key{%v}, private-key{%v}]", pubKeyString, privKeyString))
			} else {
				so.Emit("admin-remove-msg", fmt.Sprintf("removed admin [public-key{%v}, private-key{%v}]", pubKeyString, privKeyString))
			}
		})

		// Submit Forms
		so.On("submit-form", func(_type string, _address string, _description string, _specfield string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteType(_type) + lib.SERVICE.WriteAddress(_address) + lib.SERVICE.WriteDescription(_description) + lib.SERVICE.WriteSpecField(_specfield, _type) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			result := app.admin_manager.SubmitForm(str, app)
			if result.IsErr() {
				log.Println(result.Error())
				so.Emit("formID-msg", "could not submit form")
			} else {
				so.Emit("formID-msg", result.Log)
			}
		})

		// Query Forms
		so.On("query-form", func(_formID string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteFormID(_formID) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			form, err := app.admin_manager.QueryForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("form-msg", "could not find form with ID "+_formID)
			} else {
				so.Emit("form-msg", ParseForm(form))
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(_formID string, _pubkey string, _privkey string) {
			str := lib.SERVICE.WriteFormID(_formID) + lib.SERVICE.WritePubkeyString(_pubkey) + lib.SERVICE.WritePrivkeyString(_privkey)
			err := app.admin_manager.ResolveForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("resolve-msg", "could not resolve form with ID "+_formID)
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
