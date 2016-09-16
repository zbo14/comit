package app

import (
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	lib "github.com/zballs/3ii/lib"
	util "github.com/zballs/3ii/util"
	"log"
)

type ActionListener struct {
	*socketio.Server
	recvr *Switch
}

func CreateActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	recvr := StartSwitch(GenPrivKeyEd25519(), "")
	return ActionListener{server, recvr}, err
}

func FormatUpdate(update PeerMessage) string {
	str := string(update.Bytes)
	_type := lib.SERVICE.ReadField(str, "type")
	_address := lib.SERVICE.ReadField(str, "address")
	_description := lib.SERVICE.ReadField(str, "description")
	_specfield := lib.SERVICE.FieldOpts(_type).Field
	return "<strong>issue</strong> " + _type + "<br>" + "<strong>address</strong> " + _address + "<br>" + "<strong>description</strong> " + _description + "<br>" + fmt.Sprintf("<strong>%v</strong>", _specfield) + "<br><br>"
}

func (al ActionListener) BroadcastUpdates() {
	for {
		if al.recvr.IsRunning() {
			updates := al.recvr.Reactor("feed").(*MyReactor).getMsgs(byte(0x00))
			if len(updates) > 0 {
				update := updates[len(updates)-1]
				al.BroadcastTo("feed", "update-feed", FormatUpdate(update))
			}
		}
	}
}

func (al ActionListener) Run(app *Application) {

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Feed
		so.Join("feed")

		so.On("update-feed", func(update string) {
			log.Println(update)
			so.Emit("feed-udpate", update)
		})

		// Send Field Options
		so.On("select-type", func(_type string) {
			field, options := lib.SERVICE.FormatFieldOpts(_type)
			so.Emit("field-options", field, options)
		})

		// Create Accounts
		so.On("create-account", func(passphrase string) {
			pubKeyString, privKeyString, err := app.admin_manager.RegisterUser(passphrase, al.recvr)
			if err != nil {
				log.Println(err.Error())
			}
			msg := fmt.Sprintf("Your public-key is <strong><small>%v</small></strong><br>Your private-key is <strong><small>%v</small></strong><br>Do not lose it or give it to anyone! If you forget your passphrase or your account is compromised you will need your private key to regain access.", pubKeyString, privKeyString)
			so.Emit("keys-msg", msg)
		})

		// Create Admins
		so.On("create-admin", func(passphrase string) {
			pubKeyString, privKeyString, err := app.admin_manager.RegisterAdmin(passphrase, al.recvr)
			if err != nil {
				log.Println(err.Error())
				msg := fmt.Sprintf("Unauthorized")
				so.Emit("admin-keys-msg", msg)
			} else {
				msg := fmt.Sprintf("Your public-key is <strong>%v</strong><br>Your private-key is <strong>%v</strong><br>Do not lose it or give it to anyone! If you forget your passphrase or your account is compromised you will need your private key to regain access.", pubKeyString, privKeyString)
				so.Emit("admin-keys-msg", msg)
			}
		})

		// Remove Accounts
		so.On("remove-account", func(pubKeyString string, passphrase string) {
			err := app.admin_manager.RemoveUser(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("remove-msg", fmt.Sprintf("could not remove account [public-key{<strong>%v</strong>}, passphrase {<strong>%v</strong>}]", pubKeyString, passphrase))
			} else {
				so.Emit("remove-msg", fmt.Sprintf("removed account [public-key{<strong>%v</strong>}, passphrase{<strong>%v</strong>}]", pubKeyString, passphrase))
			}
		})

		// Remove Admins
		so.On("remove-admin", func(pubKeyString string, passphrase string) {
			err := app.admin_manager.RemoveAdmin(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-remove-msg", fmt.Sprintf("failed to remove admin [public-key{<strong>%v</strong>}, passphrase{<strong>%v</strong>}]", pubKeyString, passphrase))
			} else {
				so.Emit("admin-remove-msg", fmt.Sprintf("removed admin [public-key{<strong>%v</strong>}, passphrase{<strong>%v</strong>}]", pubKeyString, passphrase))
			}
		})

		// Submit Forms
		so.On("submit-form", func(_type string, _address string, _description string, _specfield string, pubKeyString string, passphrase string) {
			str := lib.SERVICE.WriteField(_type, "type") + lib.SERVICE.WriteField(_address, "address") + lib.SERVICE.WriteField(_description, "description") + lib.SERVICE.WriteSpecField(_specfield, _type) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			res := app.admin_manager.SubmitForm(str, app)
			if res.IsErr() {
				log.Println(res.Error())
				so.Emit("formID-msg", "failed to submit form")
			} else {
				so.Emit("formID-msg", res.Log)
			}
		})

		// Find Forms
		so.On("find-form", func(formID string, pubKeyString string, passphrase string) {
			str := util.WriteFormID(formID) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			form, err := app.admin_manager.FindForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("form-msg", fmt.Sprintf("failed to find form with ID <strong>%v</strong>", formID))
			} else {
				so.Emit("form-msg", ParseForm(form))
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(formID string, pubKeyString string, passphrase string) {
			str := util.WriteFormID(formID) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			err := app.admin_manager.ResolveForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("resolve-msg", fmt.Sprintf("failed to resolve form with ID <strong>%v</strong>", formID))
			} else {
				so.Emit("resolve-msg", fmt.Sprintf("resolved form with ID <strong>%v</strong>", formID))
			}
		})

		// Search forms
		so.On("search-forms", func(_type string, _address string, _specfield string, _status string, pubKeyString string, passphrase string) {
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
			str += util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			formlist, err := app.admin_manager.SearchForms(str, _status, app.cache)
			if err != nil || len(formlist) == 0 {
				log.Println(err)
				so.Emit("forms-msg", "failed to find forms")
			} else {
				var msg string = ""
				for _, form := range formlist {
					msg += ParseForm(form)
				}
				so.Emit("forms-msg", msg)
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
