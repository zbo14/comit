package actions

import (
	"bytes"
	socketio "github.com/googollee/go-socket.io"
	. "github.com/tendermint/go-common"
	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-p2p"
	wire "github.com/tendermint/go-wire"
	"github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	ntwk "github.com/zballs/3ii/network"
	sm "github.com/zballs/3ii/state"
	"github.com/zballs/3ii/types"
	. "github.com/zballs/3ii/util"
	"log"
	"runtime"
	"time"
)

type ActionListener struct {
	*socketio.Server
}

func CreateActionListener() (ActionListener, error) {
	server, err := socketio.NewServer(nil)
	return ActionListener{server}, err
}

func (al ActionListener) Run(app_ *app.App, network *p2p.Switch, peerAddr string) {

	al.On("connection", func(so socketio.Socket) {

		log.Println("connected")

		// Connect to p2p network
		so.On("connect-to-network", func(pubKeyString, privKeyString string) {

			// Create tx
			var tx = types.Tx{}
			tx.Data = []byte{sm.ConnectAccount}

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("connect-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// Check if already connected
			peer := network.Peers().Get(pubKeyString)
			if peer.IsRunning() {
				log.Panic("Error: peer already connected network") //for now
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
			addr := pubKey.Address()
			seq, err := app_.GetSequence(addr)
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

				// Peer switch info
				peer_sw := p2p.NewSwitch(ntwk.Config)
				peer_sw.SetNodeInfo(&p2p.NodeInfo{
					Network: "testing",
					Version: "311.311.311",
				})
				peer_sw.SetNodePrivKey(privKey)

				// Add reactors
				depts := network.Reactor("depts").(*ntwk.MyReactor)
				peer_sw.AddReactor("depts", depts)
				admins := network.Reactor("admins").(*ntwk.MyReactor)
				peer_sw.AddReactor("admins", admins)

				// Add listener
				l := p2p.NewDefaultListener("tcp", peerAddr, false)
				peer_sw.AddListener(l)
				peer_sw.Start()

				// Add peer to network
				addr := p2p.NewNetAddressString(peerAddr)
				_, err := network.DialPeerWithAddress(addr)

				if err != nil {
					log.Println(err.Error())
					so.Emit("connect-msg", connect_failure)
				} else {
					so.Emit("connect-msg", "connected")
				}
			}
		})

		so.On("connect-admin-to-network", func(pubKeyString, privKeyString string) {

			// Create tx
			var tx = types.Tx{}
			tx.Data = []byte{sm.ConnectAdmin}

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("connect-admin-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("connect-admin-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			addr := pubKey.Address()
			seq, err := app_.GetSequence(addr)
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
				so.Emit("connect-admin-msg", connect_failure)
			} else {

				// Check if already connected
				peer := network.Peers().Get(pubKeyString)
				if peer.IsRunning() {
					// OK if admin already connected?
					// i.e. admin previously "connected-to-network"
					so.Emit("connect-admin-msg", "connected")
					// Or error...
					// i.e. admins cannot "connect-to-network"
					// log.Panic("Error: peer already connected to network")
				} else {

					// Peer switch info
					peer_sw := p2p.NewSwitch(ntwk.Config)
					peer_sw.SetNodeInfo(&p2p.NodeInfo{
						Network: "testing",
						Version: "311.311.311",
					})
					peer_sw.SetNodePrivKey(privKey)

					// Add reactors
					admins := network.Reactor("admins").(*ntwk.MyReactor)
					peer_sw.AddReactor("admins", admins)

					// Add listener
					l := p2p.NewDefaultListener("tcp", peerAddr, false)
					peer_sw.AddListener(l)
					peer_sw.Start()

					// Add peer to network
					addr := p2p.NewNetAddressString(peerAddr)
					_, err := network.DialPeerWithAddress(addr)

					if err != nil {
						log.Println(err.Error())
						so.Emit("connect-admin-msg", connect_failure)
					} else {
						so.Emit("connect-admin-amsg", "connected")
					}
				}
			}
		})

		// Update feed
		so.On("update-feed", func() {
			depts := network.Reactor("depts").(*ntwk.MyReactor)
			for dept, chID := range app_.DeptChIDs() {
				go func(dept string, chID byte) {
					// log.Printf("DEPT %v, CH %X", dept, chID)
					idx := -1
				FOR_LOOP:
					for {
						msg := depts.GetLatestMsg(chID)
						if msg.Counter == idx {
							runtime.Gosched()
							continue FOR_LOOP
						}
						idx = msg.Counter
						var form lib.Form
						err := wire.ReadBinaryBytes(msg.Bytes[2:], &form)
						if err != nil {
							log.Println("ERROR " + err.Error())
							continue FOR_LOOP
						}
						log.Println(Fmt("%v-update", dept))
						so.Emit(Fmt("%v-update", dept), (&form).Summary())
					}
				}(dept, chID)
			}
			// Wait
			TrapSignal(func() {
				// Cleanup
			})
		})

		so.On("update-messages", func(pubKeyString string) {
			chID := app_.AdminChID(pubKeyString)
			admins := network.Reactor("admins").(*ntwk.MyReactor)
			go func() {
				idx := -1
				var message string
				for {
					msg := admins.GetLatestMsg(chID)
					if msg.Counter == idx {
						time.Sleep(time.Second * 30)
						continue
					}
					idx = msg.Counter
					err := wire.ReadBinaryBytes(msg.Bytes[2:], &message)
					if err != nil {
						log.Println("ERROR " + err.Error())
						continue
					}
					log.Printf("MESSAGE for %X\n", pubKeyString)
					so.Emit(Fmt("%X-update", pubKeyString), "<li>"+message+"</li>")
				}
			}()
			// Wait
			TrapSignal(func() {
				// Cleanup
			})
		})

		so.On("send-message", func(message, pubKeyString, adminPubKeyString string) {

			// Should already be connected to network

			// Get peer
			peer := network.Peers().Get(pubKeyString)
			if peer == nil {
				log.Panic("Error: could not find peer") //for now
			}

			// Get channel ID
			chID := app_.AdminChID(adminPubKeyString)
			if chID == byte(0) {
				so.Emit("send-message-msg", Fmt(find_admin_failure, adminPubKeyString))
				log.Println("Error: could not find admin")
				// log.Panic("Error: could not find admin")
			} else {
				// Send message
				success := peer.Send(chID, []byte(message))
				if !success {
					so.Emit("send-message-msg", Fmt(send_message_failure, adminPubKeyString))
				} else {
					so.Emit("send-message-msg", Fmt(send_message_success, adminPubKeyString))
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

			// Secret
			tx.Data = []byte(secret)

			// Set Sequence, Account, Signature
			pubKey, privKey := CreateKeys(tx.Data) // create keys now
			tx.SetSequence(0)
			tx.SetAccount(pubKey)
			tx.SetSignature(privKey, app_.GetChainID())

			log.Println(tx)

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

		// Create admin
		so.On("create-admin", func(secret, pubKeyString, privKeyString string) {

			// Create Tx
			var tx = types.Tx{}
			tx.Type = types.CreateAdminTx

			// Secret
			tx.Data = []byte(secret) // create keys later

			// PubKey
			var pubKey = crypto.PubKeyEd25519{}
			pubKeyBytes, err := HexStringToBytes(pubKeyString)
			if err != nil {
				so.Emit("create-admin-msg", Fmt(invalid_hex, pubKeyString))
				log.Panic(err)
			}
			copy(pubKey[:], pubKeyBytes[:])

			// PrivKey
			var privKey = crypto.PrivKeyEd25519{}
			privKeyBytes, err := HexStringToBytes(privKeyString)
			if err != nil {
				so.Emit("create-admin-msg", Fmt(invalid_hex, privKeyString))
				log.Panic(err)
			}
			copy(privKey[:], privKeyBytes[:])

			// Set Sequence, Account, Signature
			addr := pubKey.Address()
			seq, err := app_.GetSequence(addr)
			if err != nil {
				so.Emit("create-admin-msg", create_admin_failure)
				log.Panic(err)
			}
			tx.SetSequence(seq)
			tx.SetAccount(pubKey)
			tx.SetSignature(privKey, app_.GetChainID())

			log.Println(tx)

			// TxBytes in AppendTx request
			txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
			wire.WriteBinary(&tx, txBuf, &n, &err)
			res := app_.AppendTx(txBuf.Bytes())

			if res.IsErr() {
				so.Emit("create-admin-msg", create_admin_failure)
				log.Println(res.Error())
			} else {
				err = wire.ReadBinaryBytes(res.Data, &privKey)
				if err != nil {
					so.Emit("create-admin-msg", create_admin_failure) // for now
					log.Println(res.Error())
				}
				pubKeyString = privKey.PubKey().KeyString()
				privKeyString = BytesToHexString(privKey[:])
				msg := Fmt(create_admin_success, pubKeyString, privKeyString)
				so.Emit("create-admin-msg", msg)
				log.Printf("SUCCESS created admin with pubKey: %v", pubKeyString)
				app_.AddAdmin(pubKeyString)
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
			addr := pubKey.Address()
			seq, err := app_.GetSequence(addr)
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

			if res.IsErr() {
				so.Emit("submit-form-msg", submit_form_failure)
				log.Println(res.Error())
			} else {
				formID := BytesToHexString(res.Data)
				so.Emit("submit-form-msg", Fmt(submit_form_success, formID))
				log.Printf("SUCCESS submitted form with ID: %v", formID)
				/* Submitting forms off the network
				if network != nil {
					peer := network.Peers().Get(pubKeyString)
					if peer == nil {
						log.Panic("Error: could not find peer")
					}
					res = app_.QueryByKey(res.Data)
					if res.IsErr() {
						log.Panic(err)
					}
					var form lib.Form
					err = wire.ReadBinaryBytes(res.Data, &form)
					if err != nil {
						log.Panic(err)
					}
					dept := lib.SERVICE.ServiceDept(form.Service)
					chID := app_.DeptChID(dept)
					peer.Send(chID, res.Data)
				}
				*/
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
			addr := pubKey.Address()
			seq, err := app_.GetSequence(addr)
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

			// Query
			res := app_.QueryByKey(key)

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
		so.On("search-forms", func(before, after, service, status string) { //location

			var filters []string
			if len(service) > 0 {
				filters = append(filters, service)
			}
			// location
			if len(status) > 0 {
				filters = append(filters, status)
			}

			beforeDate := ParseTimeString(before)
			afterDate := ParseTimeString(after)

			fun1 := app_.FilterFunc(filters)
			fun2 := func(data []byte) bool {
				key, _, _ := wire.GetByteSlice(data)
				datestr := string(lib.XOR(key, service)) //location too
				date := ParseDateString(datestr)
				if !date.Before(beforeDate) || !date.After(afterDate) {
					return false
				}
				return true
			}

			in := make(chan []byte)
			out := make(chan []byte)
			// errs := make(chan error)

			go app_.Iterate(fun1, in) //errs
			go app_.IterateNext(fun2, in, out)

			var form lib.Form

			for {
				/*
					case err, more := <-errs:
						if more {
							log.Println(err.Error()) //for now
						} else {
							continue FOR_LOOP
						}
				*/
				key, more := <-out
				if more {
					log.Printf("%X\n", key)
					res := app_.QueryByKey(key)
					if res.IsErr() {
						log.Println(res.Error())
						continue
					}
					err := wire.ReadBinaryBytes(res.Data, &form)
					if err != nil {
						log.Println(err.Error())
						continue
					}
					so.Emit("search-forms-msg", (&form).Summary())
				} else {
					break
				}
			}
			log.Println("Search finished")
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
