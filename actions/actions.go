package actions

import (
	"bytes"
	"errors"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-common"
	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-p2p"
	wire "github.com/tendermint/go-wire"
	"github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	ntwk "github.com/zballs/3ii/network"
	// sm "github.com/zballs/3ii/state"
	"github.com/zballs/3ii/types"
	. "github.com/zballs/3ii/util"
	"log"
)

type ActionListener struct {
	*socketio.Server
}

func StartActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	return ActionListener{server}, err
}

func (al ActionListener) UpdateFeed(sw *p2p.Switch) {
	deptFeed := sw.Reactor("dept-feed").(*ntwk.MyReactor)
	log.Println(deptFeed.GetChannels())
	for dept, chID := range ntwk.DeptChannelIDs {
		go func(dept string, chID byte) {
			log.Printf("DEPT %v, CH %X", dept, chID)
			idx := -1
			for {
				msg := deptFeed.GetLatestMsg(chID)
				if msg.Counter == idx || msg.Bytes == nil {
					// time.Sleep?
					continue
				}
				idx = msg.Counter
				var form lib.Form
				err := wire.ReadBinaryBytes(msg.Bytes[2:], &form)
				if err != nil {
					log.Println("ERROR " + err.Error())
					continue
				}
				al.BroadcastTo(
					"feed",
					Fmt("%v-update", dept),
					(&form).Summary())
				// time.Sleep?
			}
		}(dept, chID)
	}
	// Wait
	TrapSignal(func() {
		// Cleanup
	})
}

func (al ActionListener) SendMsg(app_ *app.App, peer *p2p.Peer, key []byte) error {
	query := append([]byte{0x02}, key...)
	res := app_.Query(query)
	if res.IsErr() {
		return errors.New(res.Error())
	}
	var form lib.Form
	formBytes := res.Data
	err := wire.ReadBinaryBytes(formBytes, &form)
	if err != nil {
		return errors.New("Error decoding form bytes")
	}
	dept := lib.SERVICE.ServiceDept(form.Service)
	deptChID := byte(ntwk.DeptChannelID(dept))
	peer.Send(deptChID, formBytes)
	return nil
}

func (al ActionListener) Run(app_ *app.App, feed *p2p.Switch, peerAddr string) {

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Connect to p2p network
		so.On("connect-to-network", func(pubKeyString, privKeyString string) {

			// Create tx
			var tx = types.Tx{}

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("connect-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// Check if already connected
			err = feed.FilterConnByPubKey(pubKey)
			if err != nil {
				so.Emit("connect-msg", connect_failure)
				log.Panic(err)
			}

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("connect-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			seq, err := app_.GetSequence(pubKey.Address())
			if err != nil {
				log.Println(err.Error()) //for now
			}
			tx.SetSequence(seq)
			tx.SetAccount(pubKey)
			tx.SetSignature(privKey, app_.GetChainID())

			// TxBytes in CheckTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.CheckTx(txBuf.Bytes())

			if res.IsErr() {
				log.Println(res.Error())
				so.Emit("connect-msg", connect_failure)
			} else {
				// Create/Start peer switch
				peer_sw := p2p.NewSwitch(ntwk.Config)
				peer_sw.SetNodeInfo(&p2p.NodeInfo{
					Network: "testing",
					Version: "311.311.311",
				})
				peer_sw.SetNodePrivKey(privKey)
				l := p2p.NewDefaultListener("tcp", peerAddr, false)
				peer_sw.AddReactor("dept-feed", ntwk.DeptFeed)
				peer_sw.AddListener(l)
				peer_sw.Start()

				// Add peer to feed
				addr := p2p.NewNetAddressString(peerAddr)
				_, err := feed.DialPeerWithAddress(addr)

				if err != nil {
					log.Println(err.Error())
					so.Emit("connect-msg", connect_failure)
				} else {
					so.Join("feed")
					so.Emit("connect-msg", "connected")
					al.UpdateFeed(peer_sw)
				}
			}
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
				msg.WriteString(Fmt(select_option, service, service))
			}
			so.Emit("services", msg.String())
		})

		// Create Account
		so.On("create-account", func(secret string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.CreateAccountTx
			tx.Input.Sequence = 1

			// Secret
			tx.Data = []byte(secret)

			// Set Account, Signature
			pubKey, privKey := CreateKeys(tx.Data)
			tx.SetAccount(pubKey)
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

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("remove-account-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("remove-account-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			seq, err := app_.GetSequence(pubKey.Address())
			if err != nil {
				so.Emit("remove-account-msg", Fmt(remove_account_failure, pubKeyString)) // for now
				log.Panic(err)
			}
			tx.SetSequence(seq)
			tx.SetAccount(pubKey)
			tx.SetSignature(privKey, app_.GetChainID())

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

			// Service request information
			var buf bytes.Buffer
			buf.WriteString(lib.SERVICE.WriteField(service, "service"))
			buf.WriteString(lib.SERVICE.WriteField(address, "address"))
			buf.WriteString(lib.SERVICE.WriteField(description, "description"))
			buf.WriteString(lib.SERVICE.WriteDetail(detail, service))
			tx.Data = buf.Bytes()

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("submit-form-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("submit-form-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			seq, err := app_.GetSequence(pubKey.Address())
			if err != nil {
				log.Panic(err)
			}
			tx.SetSequence(seq)
			tx.SetAccount(pubKey)
			tx.SetSignature(privKey, app_.GetChainID())

			// TxBytes in AppendTx
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsOK() {
				formID := BytesToHexString(res.Data)
				so.Emit("submit-form-msg", Fmt(submit_form_success, formID))
				log.Printf("SUCCESS submitted form with ID: %v", formID)
				peer := feed.Peers().Get(pubKeyString)
				if peer != nil {
					err := al.SendMsg(app_, peer, res.Data)
					if err != nil {
						log.Println(err.Error())
					}
				}
			} else if res.Log == ExtractText(form_already_exists) {
				so.Emit("submit-form-msg", form_already_exists)
				log.Println(res.Error())
			} else {
				so.Emit("submit-form-msg", submit_form_failure)
				log.Println(res.Error())
			}
		})

		// Resolve Forms
		so.On("resolve-form", func(formID, pubKeyString, privKeyString string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.ResolveTx

			// FormID
			formID_bytes, err := HexStringToBytes(formID)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, formID))
				log.Panic(err)
			}
			tx.Data = formID_bytes

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("resolve-form-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			seq, err := app_.GetSequence(pubKey.Address())
			if err != nil {
				log.Panic(err)
			}
			tx.SetSequence(seq)
			tx.SetAccount(pubKey)
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

		/// Search forms
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

		/*
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
