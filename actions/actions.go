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
}

func StartActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	recvr := CreateSwitch(GenPrivKeyEd25519())
	AddListener(recvr, RecvrListenerAddr())
	AddReactor(recvr, DeptChannelIDs(), "dept-feed")
	AddReactor(recvr, ServiceChannelIDs(), "service-feed")
	recvr.Start()
	return ActionListener{server, recvr}, err
}

func FormatForm(form *Form) string {
	posted := util.ToTheMinute(form.Time().String())
	status := CheckStatus(form.Resolved())
	sd := lib.SERVICE.ServiceDetail(form.Service())
	detail := "no options"
	if sd != nil {
		detail = sd.Detail()
	}
	return "<li>" + fmt.Sprintf(line, "posted", posted) +
		fmt.Sprintf(line, "service", form.Service()) +
		fmt.Sprintf(line, "address", form.Address()) +
		fmt.Sprintf(line, "description", form.Description()) +
		fmt.Sprintf(line, detail, form.Detail()) +
		fmt.Sprintf(line, "user", form.Pubkey()) +
		fmt.Sprintf(line, "status", status) + "<br></li>"
}

func FormatUpdate(update string) string {
	action := lib.SERVICE.ReadField(update, "action")
	ID := lib.SERVICE.ReadField(update, "ID")
	service := lib.SERVICE.ReadField(update, "service")
	address := lib.SERVICE.ReadField(update, "address")
	pubkey := util.ReadPubKeyString(update)
	return "<li>" + fmt.Sprintf(line, action, ID) +
		fmt.Sprintf(line, "service", service) +
		fmt.Sprintf(line, "address", address) +
		fmt.Sprintf(line, "user", pubkey) + "<br></li>"
}

func (al ActionListener) FeedUpdates() {
	for {
		if al.recvr.IsRunning() {
			deptFeed := al.recvr.Reactor("dept-feed").(*MyReactor)
			for dept, chID := range DeptChannelIDs() {
				update := string(deptFeed.GetMsg(chID).Bytes)
				if len(update) > 0 {
					al.BroadcastTo("feed", fmt.Sprintf("%v-update", dept), FormatUpdate(update))
				}
			}
			serviceFeed := al.recvr.Reactor("service-feed").(*MyReactor)
			for service, chID := range ServiceChannelIDs() {
				update := string(serviceFeed.GetMsg(chID).Bytes)
				if len(update) > 0 {
					al.BroadcastTo("feed", fmt.Sprintf("%v-update", service), FormatUpdate(update))
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
		so.On("create-user", func(passphrase string) {
			pubKeyString, privKeyString, err := app.UserManager().Register(passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("user-keys-msg", create_user_failure)
			}
			msg := fmt.Sprintf(keys_cautionary, pubKeyString, privKeyString)
			so.Emit("user-keys-msg", msg)
		})

		// Create Admins
		so.On("create-admin", func(dept string, position string, passphrase string) {
			pubKeyString, privKeyString, err := app.AdminManager().Register(dept, position, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-keys-msg", unauthorized)
			} else {
				msg := fmt.Sprintf(keys_cautionary, pubKeyString, privKeyString)
				so.Emit("admin-keys-msg", msg)
			}
		})

		// Remove Accounts
		so.On("remove-user", func(pubKeyString string, passphrase string) {
			err := app.UserManager().Remove(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("remove-msg", fmt.Sprintf(remove_user_failure, pubKeyString, passphrase))
			} else {
				so.Emit("remove-msg", fmt.Sprintf(remove_user_success, pubKeyString, passphrase))
			}
		})

		// Remove Admins
		so.On("remove-admin", func(pubKeyString string, passphrase string) {
			err := app.AdminManager().Remove(pubKeyString, passphrase)
			if err != nil {
				log.Println(err.Error())
				so.Emit("admin-remove-msg", fmt.Sprintf(remove_admin_failure, pubKeyString, passphrase))
			} else {
				so.Emit("admin-remove-msg", fmt.Sprintf(remove_admin_success, pubKeyString, passphrase))
			}
		})

		// Submit Forms
		so.On("submit-form", func(service string, address string, description string, detail string, pubKeyString string, passphrase string) {
			err := app.UserManager().Authorize(pubKeyString, passphrase)
			if err != nil {
				so.Emit("formID-msg", unauthorized)
			} else {
				str := lib.SERVICE.WriteField("submit", "action") +
					lib.SERVICE.WriteField(service, "service") +
					lib.SERVICE.WriteField(address, "address") +
					lib.SERVICE.WriteField(description, "description") +
					lib.SERVICE.WriteDetail(detail, service) +
					util.WritePubKeyString(pubKeyString)
				result := app.AppendTx([]byte(str))
				if result.IsOK() {
					so.Emit("formID-msg", fmt.Sprintf(submit_form_success, result.Log))
					serviceChID := ServiceChannelID(service)
					deptChID := DeptChannelID(lib.SERVICE.ServiceDept(service))
					go app.UserManager().Broadcast(pubKeyString, string(result.Data), serviceChID, deptChID)
				} else if result.Log == util.ExtractText(form_already_exists) {
					so.Emit("formID-msg", form_already_exists)
				} else {
					so.Emit("formID-msg", submit_form_failure)
				}
			}
		})

		// Find Forms
		so.On("find-form", func(formID string, pubKeyString string, passphrase string) {
			err := app.UserManager().Authorize(pubKeyString, passphrase)
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
			err := app.AdminManager().Authorize(pubKeyString, passphrase)
			if err != nil {
				so.Emit("resolve-msg", unauthorized)
			} else {
				str := lib.SERVICE.WriteField("resolve", "action") +
					lib.SERVICE.WriteField(formID, "ID") +
					util.WritePubKeyString(pubKeyString)
				result := app.AppendTx([]byte(str))
				if !result.IsOK() {
					log.Println(result.Error())
					so.Emit("resolve-msg", fmt.Sprintf(resolve_form_failure, formID))
				} else {
					so.Emit("resolve-msg", fmt.Sprintf(resolve_form_success, formID))
					service := lib.SERVICE.ReadField(string(result.Data), "service")
					serviceChID := ServiceChannelID(service)
					dept := lib.SERVICE.ServiceDept(service)
					deptChID := DeptChannelID(dept)
					app.AdminManager().Broadcast(pubKeyString, string(result.Data), serviceChID, deptChID)
				}
			}
		})

		// Search forms
		so.On("search-forms", func(before string, after string, service string, address string, status string, pubKeyString string, passphrase string) {
			err := app.UserManager().Authorize(pubKeyString, passphrase)
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

		// Metrics
		so.On("calculate", func(metric string, category string, values []string, pubKeyString string, passphrase string) {
			err := app.UserManager().Authorize(pubKeyString, passphrase)
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
