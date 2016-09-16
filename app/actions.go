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

func FormatUpdate(peer_msg PeerMessage) string {
	str := string(peer_msg.Bytes)
	_type := lib.SERVICE.ReadField(str, "type")
	_address := lib.SERVICE.ReadField(str, "address")
	_description := lib.SERVICE.ReadField(str, "description")
	_specfield := lib.SERVICE.FieldOpts(_type).Field
	return "<li><strong>issue: </strong> " + _type + "<br>" + "<strong>address: </strong> " + _address + "<br>" + "<strong>description: </strong> " + _description + "<br>" + fmt.Sprintf("<strong>%v: </strong>", _specfield) + "</li><br><br>"
}

func (al ActionListener) BroadcastUpdates() {
	feed := al.recvr.Reactor("feed").(*MyReactor)
	for {
		if al.recvr.IsRunning() {
			for key, val := range DeptChannelIDs {
				peer_msg := feed.getMsg(val)
				if len(peer_msg.Bytes) > 0 {
					update := FormatUpdate(peer_msg)
					al.BroadcastTo("feed", fmt.Sprintf("%v-update", key), update)
				}
			}
		}
	}
}

func (al ActionListener) Run(app *Application) {

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Feed
		so.Join("feed")

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
			msg := fmt.Sprintf(keys_cautionary, pubKeyString, privKeyString)
			so.Emit("keys-msg", msg)
		})

		// Create Admins
		so.On("create-admin", func(passphrase string) {
			pubKeyString, privKeyString, err := app.admin_manager.RegisterAdmin(passphrase, al.recvr)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-keys-msg", "Unauthorized")
			} else {
				msg := fmt.Sprintf(keys_cautionary, pubKeyString, privKeyString)
				so.Emit("admin-keys-msg", msg)
			}
		})

		// Remove Accounts
		so.On("remove-account", func(pubKeyString string, passphrase string) {
			err := app.admin_manager.RemoveUser(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("remove-msg", fmt.Sprintf(account_remove_failure, pubKeyString, passphrase))
			} else {
				so.Emit("remove-msg", fmt.Sprintf(account_remove_success, pubKeyString, passphrase))
			}
		})

		// Remove Admins
		so.On("remove-admin", func(pubKeyString string, passphrase string) {
			err := app.admin_manager.RemoveAdmin(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-remove-msg", fmt.Sprintf(admin_remove_failure, pubKeyString, passphrase))
			} else {
				so.Emit("admin-remove-msg", fmt.Sprintf(admin_remove_success, pubKeyString, passphrase))
			}
		})

		// Submit Forms
		so.On("submit-form", func(_type string, _address string, _description string, _specfield string, pubKeyString string, passphrase string) {
			str := lib.SERVICE.WriteField(_type, "type") + lib.SERVICE.WriteField(_address, "address") + lib.SERVICE.WriteField(_description, "description") + lib.SERVICE.WriteSpecField(_specfield, _type) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			chID := DeptChannelIDs[lib.ServiceDepts[_type]]
			res := app.admin_manager.SubmitForm(str, chID, app)
			if res.IsErr() {
				log.Println(res.Error())
				so.Emit("formID-msg", "Failed to submit form")
			} else {
				so.Emit("formID-msg", fmt.Sprintf(formID, res.Log))
			}
		})

		// Find Forms
		so.On("find-form", func(formID string, pubKeyString string, passphrase string) {
			str := util.WriteFormID(formID) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			form, err := app.admin_manager.FindForm(str, app.cache)
			if err != nil {
				log.Println(err.Error())
				so.Emit("form-msg", fmt.Sprintf(find_form_failure, formID))
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
				so.Emit("resolve-msg", fmt.Sprintf(resolve_form_failure, formID))
			} else {
				so.Emit("resolve-msg", fmt.Sprintf(resolve_form_success, formID))
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
				so.Emit("forms-msg", "Failed to find forms")
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
