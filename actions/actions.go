package actions

import (
	"bytes"
	"fmt"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-crypto"
	. "github.com/tendermint/go-p2p"
	. "github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	. "github.com/zballs/3ii/network"
	. "github.com/zballs/3ii/types"
	util "github.com/zballs/3ii/util"
	"log"
)

type ActionListener struct {
	*socketio.Server
	recvr *Switch
	sendr *Switch
}

func StartActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	recvr := CreateSwitch(GenPrivKeyEd25519(), "recvr")
	AddListener(recvr, RecvrListenerAddr())
	AddReactor(recvr, FeedChannelIDs, "feed")
	sendr := CreateSwitch(GenPrivKeyEd25519(), "sendr")
	AddReactor(sendr, AdminChannelIDs, "admin")
	recvr.Start()
	sendr.Start()
	return ActionListener{server, recvr, sendr}, err
}

func FormatForm(form *Form) string {
	posted := util.ToTheMinute((*form).Time.String())
	status := CheckStatus((*form).Resolved)
	sd := lib.SERVICE.ServiceDetail((*form).Service)
	detail := "no options"
	if sd != nil {
		detail = sd.Detail()
	}
	return "<li>" + fmt.Sprintf(line, "posted", posted) + fmt.Sprintf(line, "issue", (*form).Service) + fmt.Sprintf(line, "address", (*form).Address) + fmt.Sprintf(line, "description", (*form).Description) + fmt.Sprintf(line, detail, (*form).Detail) + fmt.Sprintf(line, "pubkey", (*form).Pubkey) + fmt.Sprintf(line, "status", status) + "</li><br>"
}

func FormatUpdate(peer_msg PeerMessage) string {
	str := string(peer_msg.Bytes)
	service := lib.SERVICE.ReadField(str, "service")
	address := lib.SERVICE.ReadField(str, "address")
	description := lib.SERVICE.ReadField(str, "description")
	fieldOpts := lib.SERVICE.ServiceDetail(service)
	field := "no options"
	option := ""
	if fieldOpts != nil {
		field = fieldOpts.Detail()
		option = lib.SERVICE.ReadDetail(str, service)
	}
	return "<li>" + fmt.Sprintf(line, "issue", service) + fmt.Sprintf(line, "address", address) + fmt.Sprintf(line, "description", description) + fmt.Sprintf(line, field, option) + "</li><br>"
}

func (al ActionListener) FeedUpdates() {
	feedReactor := al.recvr.Reactor("feed").(*MyReactor)
	for {
		for dept, chID := range FeedChannelIDs {
			if al.recvr.IsRunning() {
				peer_msg := feedReactor.GetMsg(chID)
				if len(peer_msg.Bytes) > 0 {
					// To feed
					update := FormatUpdate(peer_msg)
					al.BroadcastTo("feed", fmt.Sprintf("%v-update", dept), update)
					if al.sendr.IsRunning() && dept != "general" {
						// To admin channels
						str := string(peer_msg.Bytes)
						service := lib.SERVICE.ReadField(str, "service")
						log.Println(str)
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
		for service, chID := range AdminChannelIDs {
			if admin.IsRunning() {
				peer_msg := adminReactor.GetMsg(chID)
				if len(peer_msg.Bytes) > 0 {
					// To admins
					update := FormatUpdate(peer_msg)
					al.BroadcastTo("admin", fmt.Sprintf("%v-update", service), update)
				}
			}
		}
	}
}

// func (al ActionListener) CrossCheck()

func (al ActionListener) Run(app *Application) {

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Feed
		so.Join("feed")

		// Send values
		so.On("get-values", func(category string) {
			var msg bytes.Buffer
			if category == "services" {
				for _, service := range lib.SERVICE.Services() {
					msg.WriteString(fmt.Sprintf(select_option, service, service))
				}
			} else if category == "depts" {
				for dept, _ := range lib.SERVICE.Depts() {
					msg.WriteString(fmt.Sprintf(select_option, dept, dept))
				}
			}
			so.Emit("values", msg.String())
		})

		// Send service field options
		so.On("select-service", func(service string) {
			field, options := lib.SERVICE.FormatDetail(service)
			so.Emit("field-options", field, options)
		})

		// Send dept services
		so.On("select-dept", func(dept string) {
			var msg bytes.Buffer
			for _, service := range lib.SERVICE.DeptServices(dept) {
				msg.WriteString(fmt.Sprintf(select_option, service, service))
			}
			so.Emit("services", msg.String())
		})

		// Create Accounts
		so.On("create-account", func(passphrase string) {
			pubKeyString, privKeyString, err := app.AdminManager().RegisterUser(passphrase, al.recvr)
			if err != nil {
				log.Println(err.Error())
				so.Emit("keys-msg", create_account_failure)
			}
			msg := fmt.Sprintf(keys_cautionary, pubKeyString, privKeyString)
			so.Emit("keys-msg", msg)
		})

		// Create Admins
		so.On("create-admin", func(dept string, services []string, passphrase string) {
			pubKeyString, privKeyString, err := app.AdminManager().RegisterAdmin(dept, services, passphrase, al.recvr, al.sendr)
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
			err := app.AdminManager().RemoveUser(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("remove-msg", fmt.Sprintf(remove_account_failure, pubKeyString, passphrase))
			} else {
				so.Emit("remove-msg", fmt.Sprintf(remove_account_success, pubKeyString, passphrase))
			}
		})

		// Remove Admins
		so.On("remove-admin", func(pubKeyString string, passphrase string) {
			err := app.AdminManager().RemoveAdmin(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-remove-msg", fmt.Sprintf(remove_admin_failure, pubKeyString, passphrase))
			} else {
				so.Emit("admin-remove-msg", fmt.Sprintf(remove_admin_success, pubKeyString, passphrase))
			}
		})

		// Submit Forms
		so.On("submit-form", func(service string, address string, description string, detail string, pubKeyString string, passphrase string) {
			err := app.AdminManager().AuthorizeUser(pubKeyString, passphrase)
			if err != nil {
				so.Emit("formID-msg", unauthorized)
			} else {
				str := lib.SERVICE.WriteField(service, "service") + lib.SERVICE.WriteField(address, "address") + lib.SERVICE.WriteField(description, "description") + lib.SERVICE.WriteDetail(detail, service) + util.WritePubKeyString(pubKeyString)
				result := app.AppendTx([]byte(str))
				if result.IsOK() && app.AdminManager().UserIsRunning(pubKeyString) {
					so.Emit("formID-msg", fmt.Sprintf(submit_form_success, result.Log))
					chID := FeedChannelIDs[lib.SERVICE.ServiceDept(service)]
					go app.AdminManager().UserBroadcast(pubKeyString, str, chID)
				} else if result.Log == util.ExtractText(form_already_exists) {
					so.Emit("formID-msg", form_already_exists)
				} else {
					so.Emit("formID-msg", submit_form_failure)
				}
			}
		})

		// Find Forms
		so.On("find-form", func(formID string, pubKeyString string, passphrase string) {
			err := app.AdminManager().AuthorizeUser(pubKeyString, passphrase)
			if err != nil {
				so.Emit("form-msg", unauthorized)
			} else {
				result := app.Query([]byte(formID))
				form, err := app.Cache().FindForm(formID)
				if !result.IsOK() {
					so.Emit("form-msg", fmt.Sprintf(find_form_failure, formID))
					if err == nil {
						app.Cache().RemoveForm(formID)
					}
				} else if result.IsOK() && err == nil {
					so.Emit("form-msg", FormatForm(form))
				} else {

				}
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(formID string, pubKeyString string, passphrase string) {
			err := app.AdminManager().AuthorizeAdmin(pubKeyString, passphrase)
			if err != nil {
				so.Emit("resolve-msg", unauthorized)
			} else {
				err = app.Cache().ResolveForm(formID)
				if err != nil {
					log.Println(err.Error())
					so.Emit("resolve-msg", fmt.Sprintf(resolve_form_failure, formID))
				} else {
					so.Emit("resolve-msg", fmt.Sprintf(resolve_form_success, formID))
				}
			}
		})

		// Search forms
		so.On("search-forms", func(before string, after string, service string, address string, status string, pubKeyString string, passphrase string) {
			err := app.AdminManager().AuthorizeUser(pubKeyString, passphrase)
			if err != nil {
				so.Emit("forms-msg", unauthorized)
			} else {
				var str string = ""
				if util.ToTheHour(before) != util.ToTheHour(after) {
					str += lib.SERVICE.WriteField(util.ToTheSecond(before[:19]), "before")
					str += lib.SERVICE.WriteField(util.ToTheSecond(after[:19]), "after")
				}
				if len(service) > 0 {
					str += lib.SERVICE.WriteField(service, "service")
				}
				if len(address) > 0 {
					str += lib.SERVICE.WriteField(address, "address")
				}
				log.Println(str)
				formlist := app.Cache().SearchForms(str, status)
				if formlist == nil {
					so.Emit("forms-msg", search_forms_failure, false)
				} else {
					forms := make([]string, len(formlist))
					for idx, form := range formlist {
						forms[idx] = FormatForm(form)
					}
					so.Emit("forms-msg", forms, true)
				}
			}
		})

		so.On("find-admin", func(pubKeyString string, passphrase string) {
			admin, services, err := app.AdminManager().FindAdmin(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-msg", unauthorized, false)
			} else {
				so.Join("admin")
				so.Emit("admin-msg", services, true)
				go al.AdminUpdates(admin)
			}
		})

		// Metrics
		so.On("calculate", func(metric string, category string, values []string, pubKeyString string, passphrase string) {
			err := app.AdminManager().AuthorizeUser(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("metric-msg", unauthorized)
			} else {
				output, err := app.Cache().Calculate(metric, category, values...)
				if err != nil {
					so.Emit("metric-msg", calc_metric_failure)
				} else {
					so.Emit("metric-msg", fmt.Sprintf(calc_metric_success, metric, output))
				}
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
