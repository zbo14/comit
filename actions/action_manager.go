package actions

import (
	"bytes"
	"encoding/hex"
	ws "github.com/gorilla/websocket"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	"github.com/zballs/comit/app"
	"github.com/zballs/comit/lib"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	PUBKEY_LENGTH  = 32
	PRIVKEY_LENGTH = 64
	MOMENT_LENGTH  = 32
	FORM_ID_LENGTH = 16
)

type ActionManager struct {
	*app.App
	*Hub
}

func CreateActionManager(app_ *app.App, hub *Hub) *ActionManager {
	return &ActionManager{app_, hub}
}

// Upgrader
var upgrader = ws.Upgrader{
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

	issues := am.Issues()
	var buf bytes.Buffer
	buf.WriteString(Fmt(select_option, "", "select issue"))
	for _, issue := range issues {
		buf.WriteString(Fmt(select_option, issue, issue))
	}

	conn.WriteMessage(ws.TextMessage, buf.Bytes())
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

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// Secret
		_, secret, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(secret, buf, &n, &err)
		tx.Data = buf.Bytes()

		// Set Sequence, Account, Signature
		pubKey, privKey = CreateKeys(secret)
		tx.SetSequence(0)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		res := am.AppendTx(buf.Bytes())

		if res.IsErr() {
			log.Println(res.Error())
			conn.WriteMessage(ws.TextMessage, []byte(create_account_failure))
			continue
		}

		log.Printf("SUCCESS created account with pubKey %X\n", pubKey[:])
		msg := Fmt(create_account_success, pubKey[:], privKey[:])
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()
		return
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

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Set Sequence, Account, Signature
		seq, err := am.GetSequence(pubKey.Address())
		if err != nil {
			// Shouldn't happen
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx request
		txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(tx, txBuf, &n, &err)
		res := am.AppendTx(txBuf.Bytes())

		if res.IsErr() {
			log.Println(res.Error())
			msg := Fmt(remove_account_failure, pubKeyBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}

		log.Printf("SUCCESS removed account with pubKey %X\n", pubKeyBytes)
		msg := Fmt(remove_account_success, pubKey[:])
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()
		return
	}
}

func (am *ActionManager) Connect(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Set Sequence, Account, Signature
		addr := pubKey.Address()
		seq, err := am.GetSequence(addr)
		if err != nil {
			// Shouldn't happen
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in CheckTx request
		txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(tx, txBuf, &n, &err)
		res := am.CheckTx(txBuf.Bytes())

		if res.IsErr() {
			log.Println(res.Error())
			conn.WriteMessage(ws.TextMessage, []byte(connect_failure))
			continue
		}
		log.Println("SUCCESS connected to network")
		if !am.IsAdmin(pubKey.Address()) {
			conn.WriteMessage(ws.TextMessage, []byte("connect-constituent"))
			conn.Close()
			return
		}
		conn.WriteMessage(ws.TextMessage, []byte("connect-admin"))
		conn.Close()
		return
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

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// Form
	var form *lib.Form

	for {

		// Form information
		_, issueBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		issue := string(issueBytes)
		if !am.IsIssue(issue) {
			log.Println(err.Error())
			return
		}

		_, locationBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		location := string(locationBytes)

		_, descriptionBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		description := string(descriptionBytes)

		_, mediaBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		_, extensionBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		extension := string(extensionBytes)

		_, anonymousBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		anonymous, _ := strconv.ParseBool(string(anonymousBytes))

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// TODO: field validation
		if anonymous {
			form, err = lib.MakeAnonymousForm(
				// Text
				issue,
				location,
				description,
				// Media
				mediaBytes,
				extension)
		} else {
			pubKeyString := BytesToHexString(pubKey[:])
			form, err = lib.MakeForm(
				// Text
				issue,
				location,
				description,
				pubKeyString,
				// Media
				mediaBytes,
				extension)
		}

		if err != nil {
			log.Println(err.Error())
			return //change
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(*form, buf, &n, &err)
		tx.Data = buf.Bytes()

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		_, sendToFeedBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		sendToFeed, _ := strconv.ParseBool(string(sendToFeedBytes))

		// Set Sequence, Account, Signature
		addr := pubKey.Address()
		seq, err := am.GetSequence(addr)
		if err != nil {
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		res := am.AppendTx(buf.Bytes())

		if res.IsErr() {
			log.Println(res.Error())
			conn.WriteMessage(ws.TextMessage, []byte(submit_form_failure))
			continue
		}

		log.Printf("SUCCESS submitted form with ID %X\n", res.Data)

		// Send update to feed
		if sendToFeed {
			err = am.SendUpdate(tx.Data)
			if err != nil {
				log.Panic(err)
			}
		}

		// Send response to ws
		msg := Fmt(submit_form_success, res.Data)
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		// conn.Close()
		// return
	}
}

func (am *ActionManager) FindForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Form
	var form lib.Form

	for {
		// Form ID
		_, hexBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		formIDBytes := make([]byte, FORM_ID_LENGTH)
		n, err := hex.Decode(formIDBytes, hexBytes)
		if err != nil || n != FORM_ID_LENGTH {
			msg := Fmt(invalid_formID, hexBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
		}
		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(formIDBytes, buf, &n, &err)
		key := buf.Bytes()

		// Query
		res := am.QueryByKey(key)

		if res.IsErr() {
			log.Println(res.Error())
			msg := Fmt(find_form_failure, formIDBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}
		value := res.Data
		err = wire.ReadBinaryBytes(value, &form)
		if err != nil {
			msg := Fmt(decode_form_failure, formIDBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			log.Panic(err)
		}
		log.Printf("SUCCESS found form with ID %X", formIDBytes)
		msg := (&form).Summary("find", 0)
		log.Println(string(msg))
		conn.WriteMessage(ws.TextMessage, []byte(msg))
	}
}

func (am *ActionManager) SearchForms(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	var afterTime time.Time
	var beforeTime time.Time

	count := 1

	for {

		afterTime = time.Time{}
		beforeTime = time.Time{}

		_, afterBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		after := string(afterBytes)
		if len(after) >= MOMENT_LENGTH {
			afterTime = ParseMomentString(after)
		}

		_, beforeBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		before := string(beforeBytes)
		if len(before) >= MOMENT_LENGTH {
			beforeTime = ParseMomentString(before)
		}

		var filters []string
		var includes []bool

		_, issueBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		issue := string(issueBytes)
		if len(issue) > 0 {
			filters = append(filters, issue)
			includes = append(includes, true)
		}

		// TODO: add location

		_, statusBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		status := string(statusBytes)
		if status == "resolved" {
			filters = append(filters, "resolved")
			includes = append(includes, true)
		} else if status == "unresolved" {
			filters = append(filters, "resolved")
			includes = append(includes, false)
		}

		fun1 := am.FilterFunc(filters, includes)

		fun2 := func(data []byte) bool {
			key, _, _ := wire.GetByteSlice(data)
			minutestr := string(lib.XOR(key, issue))
			time := ParseMinuteString(minutestr)
			if len(after) >= MOMENT_LENGTH && !time.After(afterTime) {
				return false
			} else if len(before) >= MOMENT_LENGTH && !time.Before(beforeTime) {
				return false
			}
			return true
		}

		in := make(chan []byte)
		out := make(chan []byte)
		// errs := make(chan error)

		// Search pipeline
		go am.Iterate(fun1, in) //errs
		go am.IterateNext(fun2, in, out)

		var form lib.Form

		for {
			key, more := <-out
			if more {
				res := am.QueryByKey(key)
				if res.IsErr() {
					log.Println(res.Error())
					continue
				}
				err = wire.ReadBinaryBytes(res.Data, &form)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				msg := (&form).Summary("search", count)
				conn.WriteMessage(ws.TextMessage, []byte(msg))
				count++
			} else {
				break
			}
		}
		log.Println("Search finished")
	}
}

func (am *ActionManager) CheckMessages(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	for {

		var pubKey crypto.PubKeyEd25519
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			return
		}

		log.Println("Getting messages...")

		// Create client
		cli := NewClient(conn, pubKey)

		// Get messages
		go am.GetMessages(cli)

		// Write messages to ws
		done := make(chan *struct{})
		go cli.writeMessagesRoutine(done)

		<-done
		//cli.Close()
		//return
	}
}

func (am *ActionManager) SendMessage(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	for {

		message := NewMessage()

		_, recipientBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(message.recipient[:], recipientBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		_, contentBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		message.content = contentBytes

		_, senderBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(message.sender[:], senderBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		err = am.sendMessage(message)
		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte("Could not send message"))
			conn.Close()
			return
		}

		conn.WriteMessage(ws.TextMessage, []byte("Message sent!"))
		// conn.Close()
		// return
	}
}

func (am *ActionManager) UpdateFeed(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	for {

		_, issueBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		issues := strings.Split(string(issueBytes), `,`)

		var pubKey crypto.PubKeyEd25519
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		hex.Decode(pubKey[:], pubKeyBytes)

		log.Println("Updating feed...")

		// Create client
		cli := NewClient(conn, pubKey)

		// Register with feed
		am.Register(cli)

		// Write updates to ws
		done := make(chan *struct{})
		go cli.writeUpdatesRoutine(issues, done)

		<-done
	}
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

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// Secret
		_, secret, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(secret, buf, &n, &err)
		tx.Data = buf.Bytes()

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Set Sequence, Account, Signature
		addr := pubKey.Address()
		seq, err := am.GetSequence(addr)
		if err != nil {
			// Shouldn't happen
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		res := am.AppendTx(buf.Bytes())

		if res.IsErr() {
			log.Println(res.Error())
			conn.WriteMessage(ws.TextMessage, []byte(create_admin_failure))
			continue
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
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()
		return
	}
}

// Remove admin

func (am *ActionManager) RemoveAdmin(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create tx
	var tx types.Tx
	tx.Type = types.RemoveAdminTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		addr := pubKey.Address()
		seq, err := am.GetSequence(addr)
		if err != nil {
			// Shouldn't happen
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx request
		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(tx, buf, &n, &err)
		res := am.AppendTx(buf.Bytes())

		if res.IsErr() {
			log.Println(err.Error())
			msg := Fmt(remove_admin_failure, pubKeyBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}

		log.Printf("SUCCESS removed admin with pubKey %X\n", pubKeyBytes)
		msg := Fmt(remove_admin_success, pubKeyBytes)
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()
		return
	}
}

// Resolve form

func (am *ActionManager) ResolveForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Type = types.ResolveTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// Form ID
		_, hexBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		formIDBytes := make([]byte, FORM_ID_LENGTH)
		n, err := hex.Decode(formIDBytes, hexBytes)
		if err != nil || n != FORM_ID_LENGTH {
			msg := Fmt(invalid_formID, hexBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}
		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(formIDBytes, buf, &n, &err)
		tx.Data = buf.Bytes()

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		n, err = hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Set Sequence, Account, Signature
		addr := pubKey.Address()
		seq, err := am.GetSequence(addr)
		if err != nil {
			log.Panic(err)
		}
		tx.SetSequence(seq)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.GetChainID())

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		key := buf.Bytes()
		res := am.AppendTx(key)

		if res.IsErr() {
			log.Println(res.Error())
			msg := Fmt(resolve_form_failure, formIDBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}

		log.Printf("SUCCESS resolved form with ID %X\n", formIDBytes)

		// Send update to feed
		am.SendUpdate(res.Data)

		// Send response to ws
		msg := Fmt(resolve_form_success, formIDBytes)
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()

		return
	}
}
