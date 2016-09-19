package app

import (
	"bytes"
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
	recvr *Switch // recv form submissions, broadcast to feed, forward to admin channels
	sendr *Switch // broadcast form submissions to admins
}

func CreateActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	recvr := CreateSwitch(GenPrivKeyEd25519(), "recvr")
	AddReactor(recvr, FeedChannelIDs, "feed")
	sendr := CreateSwitch(GenPrivKeyEd25519(), "sendr")
	AddReactor(sendr, AdminChannelIDs, "admin")
	return ActionListener{server, recvr, sendr}, err
}

func FormatUpdate(peer_msg PeerMessage) string {
	str := string(peer_msg.Bytes)
	service := lib.SERVICE.ReadField(str, "service")
	address := lib.SERVICE.ReadField(str, "address")
	description := lib.SERVICE.ReadField(str, "description")
	field := lib.SERVICE.FieldOpts(service).Field
	option := lib.SERVICE.ReadSpecField(str, service)
	return "<li>" + fmt.Sprintf(line, "issue", service) + fmt.Sprintf(line, "address", address) + fmt.Sprintf(line, "description", description) + fmt.Sprintf(line, field, option) + "</li>"
}

func (al ActionListener) FeedUpdates() {
	feedReactor := al.recvr.Reactor("feed").(*MyReactor)
	for {
		for dept, chID := range FeedChannelIDs {
			if al.recvr.IsRunning() {
				peer_msg := feedReactor.getMsg(chID)
				if len(peer_msg.Bytes) > 0 {
					// To feed
					update := FormatUpdate(peer_msg)
					al.BroadcastTo("feed", fmt.Sprintf("%v-update", dept), update)
					if al.sendr.IsRunning() {
						// To admin channels
						str := string(peer_msg.Bytes)
						service := lib.SERVICE.ReadField(str, "service")
						al.sendr.Broadcast(AdminChannelIDs[service], str)
					}
				}
			}
		}
	}
}

func (al ActionListener) AdminUpdates(admin *Switch) {
	adminReactor := admin.Reactor("admin").(*MyReactor)
	for {
		if admin.IsRunning() {
			for service, chID := range AdminChannelIDs {
				peer_msg := adminReactor.getMsg(chID)
				if len(peer_msg.Bytes) > 0 {
					// To admins
					// log.Println("*****")
					update := FormatUpdate(peer_msg)
					al.BroadcastTo("admin", fmt.Sprintf("%v-update", service), update)
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

		// Send field options
		so.On("select-service", func(service string) {
			field, options := lib.SERVICE.FormatFieldOpts(service)
			so.Emit("field-options", field, options)
		})

		// Send services
		so.On("select-dept", func(dept string) {
			var msg bytes.Buffer
			for _, service := range lib.SERVICE.DeptServices(dept) {
				msg.WriteString(fmt.Sprintf(select_option, service, service))
			}
			so.Emit("services", msg.String())
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
		so.On("create-admin", func(dept string, services []string, passphrase string) {
			pubKeyString, privKeyString, err := app.admin_manager.RegisterAdmin(dept, services, passphrase, al.recvr, al.sendr)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-keys-msg", unauthorized)
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
		so.On("submit-form", func(service string, address string, description string, specfield string, pubKeyString string, passphrase string) {
			str := lib.SERVICE.WriteField(service, "service") + lib.SERVICE.WriteField(address, "address") + lib.SERVICE.WriteField(description, "description") + lib.SERVICE.WriteSpecField(specfield, service) + util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			chID := FeedChannelIDs[lib.SERVICE.ServiceDept(service)]
			res := app.admin_manager.SubmitForm(str, chID, app)
			if res.IsErr() {
				log.Println(res.Log)
				if res.Log == form_already_exists {
					so.Emit("formID-msg", form_already_exists)
				} else {
					so.Emit("formID-msg", submit_form_failure)
				}
			} else {
				so.Emit("formID-msg", fmt.Sprintf(submit_form_success, res.Log))
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
		so.On("search-forms", func(service string, address string, status string, pubKeyString string, passphrase string) {
			var str string = ""
			if len(service) > 0 {
				str += lib.SERVICE.WriteField(service, "service")
			}
			if len(address) > 0 {
				str += lib.SERVICE.WriteField(address, "address")
			}
			str += util.WritePubKeyString(pubKeyString) + util.WritePassphrase(passphrase)
			formlist, err := app.admin_manager.SearchForms(str, status, app.cache)
			if err != nil || len(formlist) == 0 {
				log.Println(err)
				so.Emit("forms-msg", search_forms_failure)
			} else {
				var msg string = ""
				for _, form := range formlist {
					msg += ParseForm(form)
				}
				so.Emit("forms-msg", msg)
			}
		})

		so.On("find-admin", func(pubKeyString string, passphrase string) {
			admin, services, err := app.admin_manager.FindAdmin(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-msg", find_admin_failure, false)
			} else {
				so.Join("admin")
				so.Emit("admin-msg", services, true)
				go al.AdminUpdates(admin)
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
