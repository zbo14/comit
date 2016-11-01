package actions

import (
	"bytes"
	"encoding/hex"
	"github.com/gorilla/websocket"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-p2p"
	wire "github.com/tendermint/go-wire"
	"github.com/zballs/comit/app"
	"github.com/zballs/comit/lib"
	ntwk "github.com/zballs/comit/network"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"log"
	// "runtime"
	"net/http"
	"strings"
	"time"
)

type ActionManager struct {
	app     *app.App
	network *p2p.Switch
}

func CreateActionManager(app_ *app.App, network *p2p.Switch) *ActionManager {
	return &ActionManager{
		app:     app_,
		network: network,
	}
}

// Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(req *http.Request) bool {
		return true
	},
}

// Issues
func (am *ActionManager) GetIssues(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	issues := am.app.Issues()
	var buf bytes.Buffer
	buf.WriteString(Fmt(select_option, "", "select issue"))
	for _, issue := range issues {
		buf.WriteString(Fmt(select_option, issue, issue))
	}

	conn.WriteMessage(websocket.TextMessage, buf.Bytes())
	conn.Close()
}

// Create Account
func (am *ActionManager) CreateAccount(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Type = types.CreateAccountTx

	// Secret
	_, secret, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(secret, buf, &n, &err)
	tx.Data = buf.Bytes()

	// Set Sequence, Account, Signature
	pubKey, privKey := CreateKeys(secret)
	tx.SetSequence(0)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx request
	buf = new(bytes.Buffer)
	wire.WriteBinary(tx, buf, &n, &err)
	res := am.app.AppendTx(buf.Bytes())

	if res.IsErr() {
		// log.Println(res.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(create_account_failure))
	} else {
		// log.Printf("SUCCESS created account with pubKey %X\n", pubKey[:])
		msg := Fmt(create_account_success, pubKey[:], privKey[:])
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	}
}

func (am *ActionManager) RemoveAccount(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Type = types.RemoveAccountTx

	// PubKey
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	pubKeyBytes := make([]byte, 32)
	n, err := hex.Decode(pubKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
	}
	pubKeyBytes = pubKeyBytes[:n]
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
	}
	privKeyBytes = privKeyBytes[:n]
	var privKey crypto.PrivKeyEd25519
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	seq, err := am.app.GetSequence(pubKey.Address())
	if err != nil {
		msg := Fmt(remove_account_failure, pubKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	res := am.app.AppendTx(txBuf.Bytes())

	if res.IsErr() {
		// log.Println(res.Error())
		msg := Fmt(remove_account_failure, pubKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	} else {
		// log.Printf("SUCCESS removed account with pubKey %X\n", pubKeyBytes)
		msg := Fmt(remove_account_success, pubKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	}
}

func (am *ActionManager) Connect(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx

	// PubKey
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	pubKeyBytes := make([]byte, 32)
	n, err := hex.Decode(pubKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
	}
	pubKeyBytes = pubKeyBytes[:n]
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
	}
	privKeyBytes = privKeyBytes[:n]
	var privKey crypto.PrivKeyEd25519
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	addr := pubKey.Address()
	seq, err := am.app.GetSequence(addr)
	if err != nil {
		log.Println(err.Error()) //for now
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in CheckTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	res := am.app.CheckTx(txBuf.Bytes())

	if res.IsErr() {
		log.Println(res.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(connect_failure))
		conn.Close()
	} else {

		// Check if already connected
		pubKeyString := BytesToHexString(pubKeyBytes)
		peer := am.network.Peers().Get(pubKeyString)
		if peer != nil {
			log.Println("Peer already connected")
			if !am.app.IsAdmin(pubKey.Address()) {
				conn.WriteMessage(websocket.TextMessage, []byte("connect-constituent"))
				conn.Close()
				return
			}
			conn.WriteMessage(websocket.TextMessage, []byte("connect-admin"))
			conn.Close()
			return
		}

		// Peer switch info
		peer_sw := p2p.NewSwitch(ntwk.Config)
		peer_sw.SetNodeInfo(&p2p.NodeInfo{
			Network: "testing",
			Version: "311.311.311",
		})
		peer_sw.SetNodePrivKey(privKey)

		// Add reactor
		issues := am.network.Reactor("issues").(*ntwk.MyReactor)
		peer_sw.AddReactor("issues", issues)

		peerAddr := req.RemoteAddr

		// Add listener
		l := p2p.NewDefaultListener("tcp", peerAddr, false)
		peer_sw.AddListener(l)
		peer_sw.Start()

		// Add peer to network
		addr := p2p.NewNetAddressString(peerAddr)
		_, err := am.network.DialPeerWithAddress(addr)

		if err != nil {
			log.Println(err.Error())
			conn.WriteMessage(websocket.TextMessage, []byte(connect_failure))
			conn.Close()
		} else {
			log.Println("SUCCESS peer connected to network")
			if !am.app.IsAdmin(pubKey.Address()) {
				conn.WriteMessage(websocket.TextMessage, []byte("connect-constituent"))
				conn.Close()
				return
			}
			conn.WriteMessage(websocket.TextMessage, []byte("connect-admin"))
			conn.Close()
		}
	}
}

func (am *ActionManager) SubmitForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create tx
	var tx types.Tx
	tx.Type = types.SubmitTx

	// Form information
	_, issueBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	issue := string(issueBytes)
	if !am.app.IsIssue(issue) {
		log.Panic(err) // for now
	}
	_, locationBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	location := string(locationBytes)
	_, descriptionBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	description := string(descriptionBytes)

	// TODO: field validation
	form, err := lib.MakeForm(issue, location, description)
	if err != nil {
		log.Panic(err) //for now
	}
	formBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(*form, formBuf, &n, &err)
	tx.Data = formBuf.Bytes()

	// PubKey
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	pubKeyBytes := make([]byte, 32)
	n, err = hex.Decode(pubKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		return
	}
	pubKeyBytes = pubKeyBytes[:n]
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		return
	}
	privKeyBytes = privKeyBytes[:n]
	var privKey crypto.PrivKeyEd25519
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	addr := pubKey.Address()
	seq, err := am.app.GetSequence(addr)
	if err != nil {
		log.Println(err.Error()) //for now
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx
	txBuf := new(bytes.Buffer)
	wire.WriteBinary(tx, txBuf, &n, &err)
	res := am.app.AppendTx(txBuf.Bytes())

	if res.IsErr() {
		// log.Println(res.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(submit_form_failure))
		return
	}

	msg := Fmt(submit_form_success, res.Data)
	// log.Printf("SUCCESS submitted form with ID: %X\n", res.Data)
	conn.WriteMessage(websocket.TextMessage, []byte(msg))

	// Check peer
	pubKeyString := BytesToHexString(pubKeyBytes)
	peer := am.network.Peers().Get(pubKeyString)
	if peer == nil {
		log.Println("Could not find peer")
		return
	}
	if !peer.IsRunning() {
		log.Println("Peer is not running")
		return
	}
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(res.Data, buf, &n, &err)
	key := buf.Bytes()
	res = am.app.QueryByKey(key)
	if res.IsErr() {
		log.Panic(err)
	}
	err = wire.ReadBinaryBytes(res.Data, form)
	if err != nil {
		log.Panic(err)
	}
	chID := am.app.IssueID(form.Issue)
	peer.Send(chID, res.Data)
}

func (am *ActionManager) FindForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Form ID
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	keyBytes := make([]byte, 16)
	n, err := hex.Decode(keyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
	}
	keyBytes = keyBytes[:n]
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(keyBytes, buf, &n, &err)
	key := buf.Bytes()

	// Query
	res := am.app.QueryByKey(key)

	if res.IsErr() {
		log.Println(res.Error())
		msg := Fmt(find_form_failure, keyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
	var form lib.Form
	value := res.Data
	err = wire.ReadBinaryBytes(value, &form)
	if err != nil {
		msg := Fmt(decode_form_failure, keyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		log.Panic(err)
	}
	msg := (&form).Summary()
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
	log.Printf("SUCCESS found form with ID: %X", keyBytes)
}

func (am *ActionManager) SearchForms(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	_, afterBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	after := string(afterBytes)
	afterDate := ParseMomentString(after)

	_, beforeBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	before := string(beforeBytes)
	beforeDate := ParseMomentString(before)

	var filters []string

	_, issueBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	issue := string(issueBytes)
	if len(issue) > 0 {
		filters = append(filters, issue)
	}

	// TODO: add location

	_, statusBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	status := string(statusBytes)
	if len(status) > 0 {
		filters = append(filters, status)
	}

	fun1 := am.app.FilterFunc(filters)

	fun2 := func(data []byte) bool {
		key, _, _ := wire.GetByteSlice(data)
		datestr := string(lib.XOR(key, issue))
		date := ParseDateString(datestr)
		log.Println(date.After(afterDate))
		log.Println(date.Before(beforeDate))
		if !date.After(afterDate) || !date.Before(beforeDate) {
			return false
		}
		return true
	}

	in := make(chan []byte)
	out := make(chan []byte)
	// errs := make(chan error)

	go am.app.Iterate(fun1, in) //errs
	go am.app.IterateNext(fun2, in, out)

	var form lib.Form

	for {
		key, more := <-out
		if more {
			res := am.app.QueryByKey(key)
			if res.IsErr() {
				log.Println(res.Error())
				continue
			}
			err := wire.ReadBinaryBytes(res.Data, &form)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			msg := (&form).Summary()
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
		} else {
			break
		}
	}
	log.Println("Search finished")
}

func (am *ActionManager) UpdateFeed(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	_, issueBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	issues := strings.Split(string(issueBytes), `,`)
	log.Println(issues)
	reactor := am.network.Reactor("issues").(*ntwk.MyReactor)

	for _, issue := range issues {
		chID := am.app.IssueID(issue)
		log.Printf("ISSUE %v, CH %d\n", issue, chID)
		go func(issue string, chID byte) {
			idx := -1
		FOR_LOOP:
			for {
				log.Println("Updating feed...")
				peerMsg := reactor.GetLatestMsg(chID)
				if peerMsg.Counter == idx {
					time.Sleep(time.Second * 10)
					// runtime.Gosched
					continue FOR_LOOP
				}
				idx = peerMsg.Counter
				var form lib.Form
				err := wire.ReadBinaryBytes(peerMsg.Bytes[2:], &form)
				if err != nil {
					log.Println(err.Error())
					continue FOR_LOOP
				}
				log.Printf("%s update\n", issue)
				msg := (&form).Summary()
				conn.WriteMessage(websocket.TextMessage, []byte(msg))
			}
		}(issue, chID)
	}

	// Wait
	TrapSignal(func() {
		// Cleanup
	})
}

// Create Admin
func (am *ActionManager) CreateAdmin(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Type = types.CreateAdminTx

	// Secret
	_, secret, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}

	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(secret, buf, &n, &err)
	tx.Data = buf.Bytes()

	// PubKey
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	pubKeyBytes := make([]byte, 32)
	n, err = hex.Decode(pubKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
		return
	}
	pubKeyBytes = pubKeyBytes[:n]
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		conn.Close()
		return
	}
	privKeyBytes = privKeyBytes[:n]
	var privKey crypto.PrivKeyEd25519
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	addr := pubKey.Address()
	seq, err := am.app.GetSequence(addr)
	if err != nil {
		log.Println(err.Error()) //for now
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx request
	buf = new(bytes.Buffer)
	wire.WriteBinary(tx, buf, &n, &err)
	res := am.app.AppendTx(buf.Bytes())

	if res.IsErr() {
		log.Println(res.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(create_admin_failure))
		conn.Close()
		return
	}
	pubKeyBytes, _, err = wire.GetByteSlice(res.Data)
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes, _, err = wire.GetByteSlice(res.Data)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("SUCCESS created admin with pubKey %X\n", pubKey[:])
	msg := Fmt(create_admin_success, pubKeyBytes, privKeyBytes)
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
	conn.Close()
}

// Remove admin

// Resolve form

func (am *ActionManager) ResolveForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Type = types.ResolveTx

	// Form ID
	_, rawBytes, err := conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	formID := make([]byte, 16)
	n, err := hex.Decode(formID, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		return
	}

	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(formID, buf, &n, &err)
	tx.Data = buf.Bytes()

	// PubKey
	var pubKey crypto.PubKeyEd25519
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	pubKeyBytes := make([]byte, 32)
	n, err = hex.Decode(pubKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		return
	}
	copy(pubKey[:], pubKeyBytes[:])

	// Must be connected to resolve form
	pubKeyString := BytesToHexString(pubKeyBytes)
	peer := am.network.Peers().Get(pubKeyString)
	if peer == nil {
		log.Println("Could not find peer")
		return
	}
	if !peer.IsRunning() {
		log.Println("Peer is not running")
		return
	}

	// PrivKey
	var privKey crypto.PrivKeyEd25519
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(invalid_hex))
		return
	}
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	addr := pubKey.Address()
	seq, err := am.app.GetSequence(addr)
	if err != nil {
		log.Panic(err)
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	res := am.app.AppendTx(txBuf.Bytes())

	if res.IsErr() {
		log.Println(res.Error())
		msg := Fmt(resolve_form_failure, formID)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		return
	}

	msg := Fmt(resolve_form_success, formID)
	log.Printf("SUCCESS resolved form with ID: %X\n", formID)
	conn.WriteMessage(websocket.TextMessage, []byte(msg))

	buf = new(bytes.Buffer)
	wire.WriteByteSlice(formID, buf, &n, &err)
	key := buf.Bytes()
	res = am.app.QueryByKey(key)
	if res.IsErr() {
		log.Panic(err)
	}
	var form lib.Form
	err = wire.ReadBinaryBytes(res.Data, form)
	if err != nil {
		log.Panic(err)
	}
	chID := am.app.IssueID(form.Issue)
	peer.Send(chID, res.Data)
}
