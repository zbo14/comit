package app

import (
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	// . "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	lib "github.com/zballs/3ii/lib"
	util "github.com/zballs/3ii/util"
	"log"
)

type ActionListener struct {
	*socketio.Server
	Sender   *Switch
	Receiver *Switch
}

func CreateActionListener() (ActionListener, error) {
	sender, receiver := CreateSwitchPair()
	server, err := socketio.NewServer(nil)
	return ActionListener{server, sender, receiver}, err
}

func FormatUpdate(update PeerMessage) string {
	str := string(update.Bytes)
	_type := "<strong>issue</strong>" + lib.SERVICE.ReadField(str, "type")
	_address := "<strong>address</strong>" + lib.SERVICE.ReadField(str, "address")
	_description := "<strong>description</strong>" + lib.SERVICE.ReadField(str, "description")
	// _specfield := "<strong>detail</strong>" + lib.SERVICE.ReadSpecField(str, _type)
	return _type + "<br>" + _address + "<br>" + _description + "<br>" + _specfield + "<br><br>"
}

func (al ActionListener) Run(app *Application) {
	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Send Field Options
		so.On("select-type", func(_type string) {
			field, options := lib.SERVICE.FormatFieldOpts(_type)
			so.Emit("field-options", field, options)
		})

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
		so.On("submit-form", func(_type string, _address string, _description string, _specfield string, pubKeyString string, privKeyString string) {
			str := lib.SERVICE.WriteField(_type, "type") + lib.SERVICE.WriteField(_address, "address") + lib.SERVICE.WriteField(_description, "description") + lib.SERVICE.WriteSpecField(_specfield, _type) + util.WritePubKeyString(pubKeyString) + util.WritePrivKeyString(privKeyString)
			result := app.admin_manager.SubmitForm(str, app)
			if result.IsErr() {
				log.Println(result.Error())
				so.Emit("formID-msg", "could not submit form")
			} else {
				so.Emit("formID-msg", result.Log)
				if al.Sender.IsRunning() {
					al.Sender.Broadcast(byte(0x00), util.RemovePrivKeyString(str))
				}
			}
		})

		// Find Forms
		so.On("find-form", func(_formID string, pubKeyString string, privKeyString string) {
			str := util.WriteFormID(_formID) + util.WritePubKeyString(pubKeyString) + util.WritePrivKeyString(privKeyString)
			form, err := app.admin_manager.FindForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("form-msg", "could not find form with ID "+_formID)
			} else {
				so.Emit("form-msg", ParseForm(form))
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(_formID string, pubKeyString string, privKeyString string) {
			str := util.WriteFormID(_formID) + util.WritePubKeyString(pubKeyString) + util.WritePrivKeyString(privKeyString)
			err := app.admin_manager.ResolveForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("resolve-msg", "could not resolve form with ID "+_formID)
			} else {
				so.Emit("resolve-msg", "resolved form with ID "+_formID)
			}
		})

		// Search forms
		so.On("search-forms", func(_type string, _address string, _specfield string, _status string, pubKeyString string, privKeyString string) {
			var str string = ""
			if len(_type) > 0 {
				str += lib.SERVICE.WriteField(_type, "type")
			}
			if len(_address) > 0 {
				str += lib.SERVICE.WriteField(_address, "address")
			}
			if len(_specfield) > 0 {
				str += lib.SERVICE.WriteSpecField(_specfield, _type)
			}
			str += util.WritePubKeyString(pubKeyString) + util.WritePrivKeyString(privKeyString)
			formlist, err := app.admin_manager.SearchForms(str, _status, app.cache)
			if err != nil || len(formlist) == 0 {
				log.Println(err)
				so.Emit("forms-msg", "could not find forms")
			} else {
				var msg string = ""
				for _, form := range formlist {
					msg += ParseForm(form)
				}
				so.Emit("forms-msg", msg)
			}
		})

		// Recv Feed updates
		go func() {
			if al.Receiver.IsRunning() {
				updates := al.Receiver.Reactor("feed").(*MyReactor).getMsgs(byte(0x00))
				if len(updates) > 0 {
					update := updates[len(updates)-1]
					so.Emit("update-feed", FormatUpdate(update))
				}
			}
		}()

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
