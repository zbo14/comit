package actions

import (
	"bytes"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-common"
	crypto "github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	"github.com/zballs/3ii/types"
	. "github.com/zballs/3ii/util"
	"log"
)

var pubKey crypto.PubKeyEd25519
var privKey crypto.PrivKeyEd25519

type ActionListener struct {
	*socketio.Server
}

func StartActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	return ActionListener{server}, err
}

func (al ActionListener) UpdateFeed(key []byte, app_ *app.App) (res tmsp.Result) {
	query := append([]byte{0x02}, key...)
	res = app_.Query(query)
	if res.IsErr() {
		return res
	}
	var form lib.Form
	value := res.Data
	err := wire.ReadBinaryBytes(value, &form)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog("Error decoding form: " + err.Error())
	}
	msg := (&form).Summary()
	service := form.Service
	al.BroadcastTo("feed", Fmt("%v-update", service), msg)
	dept := lib.SERVICE.ServiceDept(service)
	al.BroadcastTo("feed", Fmt("%v-update", dept), msg)
	return tmsp.OK
}

// func (al ActionListener) CrossCheck()

func (al ActionListener) Run(app_ *app.App) {

	chainID := app_.GetChainID()

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Feed
		so.Join("feed")

		// Send values

		// Send service field options
		so.On("select-service", func(service string) {
			field, options := lib.SERVICE.FormatDetail(service)
			so.Emit("field-options", field, options)
		})

		// Send dept services
		so.On("select-dept", func(dept string) {
			var msg bytes.Buffer
			for _, service := range lib.SERVICE.DeptServices(dept) {
				msg.WriteString(Fmt(select_option, service, service))
			}
			so.Emit("services", msg.String())
		})

		/*
			so.On("get-values", func(category string) {
				var msg bytes.Buffer
				if category == "services" {
					for _, service := range lib.SERVICE.Services() {
						msg.WriteString(Fmt(select_option, service, service))
					}
				} else if category == "depts" {
					for dept, _ := range lib.SERVICE.Depts() {
						msg.WriteString(Fmt(select_option, dept, dept))
					}
				}
				so.Emit("values", msg.String())
			})
		*/

		// Create Users
		so.On("create-account", func(secret string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.CreateAccountTx
			tx.Input.Sequence = int(app_.Query([]byte{0x01}).Data[1]) + 1

			// Password
			tx.Data = []byte(secret)

			// Set Account
			pubKey, privKey := CreateKeys(tx.Data)
			tx.SetAccount(pubKey)

			// Set Signature
			tx.SetSignature(privKey, app_.GetChainID())

			// TxBytes in AppendTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsErr() {
				so.Emit("create-account-msg", create_account_failure)
				log.Println(res.Error())
			} else {
				pubKeyString := BytesToHexString(pubKey[:])
				privKeyString := BytesToHexString(privKey[:])
				msg := Fmt(create_account_success, pubKeyString, privKeyString) // return address?
				so.Emit("create-account-msg", msg)
				log.Printf("SUCCESS created account with pubKey: %v", pubKeyString)
			}
		})

		so.On("remove-account", func(pubKeyString, privKeyString string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.RemoveAccountTx
			tx.Input.Sequence = int(app_.Query([]byte{0x01}).Data[1]) + 1

			// Set Account
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("remove-account-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])
			tx.SetAccount(pubKey)

			// Set Signature
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("remove-account-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])
			tx.SetSignature(privKey, chainID)

			// TxBytes in AppendTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsErr() {
				so.Emit("remove-account-msg", Fmt(remove_account_failure, pubKeyString))
				log.Println(res.Error())
			} else {
				so.Emit("remove-account-msg", Fmt(remove_account_success, pubKeyString))
				log.Printf("SUCCESS removed account with pubKey: %v", pubKeyString)
			}

		})

		// Submit Forms
		so.On("submit-form", func(service, address, description, detail, pubKeyString, privKeyString string) {

			// Create tx
			var tx = types.Tx{}
			tx.Type = types.SubmitTx
			tx.Input.Sequence = int(app_.Query([]byte{0x01}).Data[1]) + 1

			// Service request information
			var buf bytes.Buffer
			buf.WriteString(lib.SERVICE.WriteField(service, "service"))
			buf.WriteString(lib.SERVICE.WriteField(address, "address"))
			buf.WriteString(lib.SERVICE.WriteField(description, "description"))
			buf.WriteString(lib.SERVICE.WriteDetail(detail, service))
			tx.Data = buf.Bytes()

			// Account Address
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("submit-form-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])
			tx.SetAccount(pubKey)

			// Account Signature
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("submit-form-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])
			tx.SetSignature(privKey, chainID)

			// TxBytes in AppendTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsOK() {
				formID := BytesToHexString(res.Data)
				so.Emit("submit-form-msg", Fmt(submit_form_success, formID))
				al.UpdateFeed(res.Data, app_)
				log.Printf("SUCCESS submitted form with ID: %v", formID)
			} else if res.Log == ExtractText(form_already_exists) {
				so.Emit("submit-form-msg", form_already_exists)
				log.Println(res.Error())
			} else {
				so.Emit("submit-form-msg", submit_form_failure)
				log.Println(res.Error())
			}
		})

		// Find Forms

		so.On("find-form", func(formID string) {

			// FormID
			key, err := HexStringToBytes(formID)
			if err != nil {
				so.Emit("find-form-msg", Fmt(invalid_hex, formID))
				log.Panic(err)
			}
			query := append([]byte{0x02}, key...)

			// Query
			res := app_.Query(query)

			if res.IsErr() {
				so.Emit("find-form-msg", Fmt(find_form_failure, formID))
				log.Println(res.Error())
			} else {
				var form lib.Form
				value := res.Data
				err := wire.ReadBinaryBytes(value, &form)
				if err != nil {
					so.Emit("find-form-msg", Fmt(decode_form_failure, formID))
					log.Panic(err)
				}
				msg := (&form).Summary()
				so.Emit("find-form-msg", msg)
				log.Printf("SUCCESS found form with ID: %v", formID)
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(formID, pubKeyString, privKeyString string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.ResolveTx
			tx.Input.Sequence = int(app_.Query([]byte{0x01}).Data[1]) + 1

			// FormID
			formID_bytes, err := HexStringToBytes(formID)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, formID))
				log.Panic(err)
			}
			tx.Data = formID_bytes

			// Account Address
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])
			tx.SetAccount(pubKey)

			// Account Signature
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])
			tx.SetSignature(privKey, app_.GetChainID())

			// TxBytes in AppendTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsErr() {
				so.Emit("resolve-form-msg", Fmt(resolve_form_failure, formID))
				log.Println(res.Error())
			} else {
				so.Emit("resolve-form-msg", Fmt(resolve_form_success, formID))
				log.Printf("SUCCESS resolved form with ID: %v", formID)
			}
		})

		/*
			// Search forms
			so.On("search-forms", func(before string, after string, service string, address string, status string, pubKeyString string, passphrase string) {
				err := app_.UserManager().Authorize(pubKeyString, passphrase)
				if err != nil {
					so.Emit("forms-msg", unauthorized)
				} else {
					var str string = ""
					if ToTheHour(before) != ToTheHour(after) {
						str += lib.SERVICE.WriteField(ToTheSecond(before[:19]), "before")
						str += lib.SERVICE.WriteField(ToTheSecond(after[:19]), "after")
					}
					if len(service) > 0 {
						str += lib.SERVICE.WriteField(service, "service")
					}
					if len(address) > 0 {
						str += lib.SERVICE.WriteField(address, "address")
					}
					log.Println(str)
					formlist := app_.Cache().SearchForms(str, status)
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
				err := app_.UserManager().Authorize(pubKeyString, passphrase)
				if err != nil {
					log.Println(err.Error())
					so.Emit("metric-msg", unauthorized)
				} else {
					output, err := app_.Cache().Calculate(metric, category, values...)
					if err != nil {
						so.Emit("metric-msg", calc_metric_failure)
					} else {
						so.Emit("metric-msg", Fmt(calc_metric_success, metric, output))
					}
				}
			})
		*/

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
