package actions

import (
	"bufio"
	"bytes"
	"encoding/hex"
	ws "github.com/gorilla/websocket"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/forms"
	"github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"net"
	"net/http"
	"strconv"
	"strings"
	//"time"
)

const (
	PUBKEY_LENGTH  = 32
	PRIVKEY_LENGTH = 64
	MOMENT_LENGTH  = 32
	FORM_ID_LENGTH = 16

	QUERY_CHAIN_ID byte = 0
	QUERY_SIZE     byte = 1
	QUERY_BY_KEY   byte = 2
	QUERY_BY_INDEX byte = 3
	QUERY_ISSUES   byte = 4

	CHECK_TX_CONNECT  byte = 5
	CHECK_TX_REGISTER byte = 6
	CHECK_TX_INBOX    byte = 7
	CHECK_TX_UPDATE   byte = 8

	// CHECK_TX_SEND     byte = 9
	// CHECK_TX_CHECK    byte = 10
)

// TODO: store chainID, issues so we
// don't have to do queries every time

type ActionManager struct {
	ServerAddr string
	ChainID    string
	Issues     []string

	types.Logger
}

func CreateActionManager(serverAddr string) *ActionManager {

	am := &ActionManager{
		ServerAddr: serverAddr,
		Logger:     types.NewLogger("action_manager"),
	}

	am.GetChainID()
	am.GetIssues()

	return am
}

// Upgrader
var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(req *http.Request) bool {
		return true
	},
}

// Queries
func (am *ActionManager) KeyQuery(key []byte) *tmsp.Request {
	query := make([]byte, wire.ByteSliceSize(key)+1)
	buf := query
	buf[0] = QUERY_BY_KEY
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	req := tmsp.ToRequestQuery(query)
	return req
}

func (am *ActionManager) IndexQuery(i int) (*tmsp.Request, error) {
	query := make([]byte, 100)
	buf := query
	buf[0] = QUERY_BY_INDEX
	buf = buf[1:]
	n, err := wire.PutVarint(buf, i)
	if err != nil {
		return nil, err
	}
	query = query[:n+1]
	req := tmsp.ToRequestQuery(query)
	return req, nil
}

// Write, Read
func (am *ActionManager) WriteRequest(req *tmsp.Request, conn net.Conn) error {

	bufWriter := bufio.NewWriter(conn)
	err := tmsp.WriteMessage(req, bufWriter)
	if err != nil {
		return err
	}
	flush := tmsp.ToRequestFlush()
	tmsp.WriteMessage(flush, bufWriter)
	bufWriter.Flush()
	return nil
}

func (am *ActionManager) ReadResponse(conn net.Conn) *tmsp.Response {

	bufReader := bufio.NewReader(conn)
	res := &tmsp.Response{}
	err := tmsp.ReadMessage(bufReader, res)
	if err != nil {
		return nil
	}
	return res
}

func (am *ActionManager) GetChainID() {

	for {

		reqQuery := tmsp.ToRequestQuery([]byte{QUERY_CHAIN_ID})
		c, _ := net.Dial("tcp", am.ServerAddr)

		err := am.WriteRequest(reqQuery, c)

		if err != nil {
			am.Error(write_request_failure)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			am.Error(read_response_failure)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			am.Error(read_response_failure)
			continue
		}

		am.ChainID = resQuery.Log
		return
	}
}

func (am *ActionManager) GetIssues() {

	for {

		reqQuery := tmsp.ToRequestQuery([]byte{QUERY_ISSUES})
		c, _ := net.Dial("tcp", am.ServerAddr)

		err := am.WriteRequest(reqQuery, c)

		if err != nil {
			am.Error(write_request_failure)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			am.Error(read_response_failure)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			am.Error(read_response_failure)
			continue
		}

		var issues []string
		err = wire.ReadBinaryBytes(resQuery.Data, &issues)
		if err != nil {
			panic(err)
		}

		am.Issues = issues
		return
	}
}

func (am *ActionManager) SendIssues(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	buf.WriteString(Fmt(select_option, "", "select issue"))

	for _, issue := range am.Issues {
		buf.WriteString(Fmt(select_option, issue, issue))
	}

	conn.WriteMessage(ws.TextMessage, buf.Bytes())
	conn.Close()

	return
}

// Create Account
func (am *ActionManager) CreateAccount(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
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
			am.Error(err.Error())
			return
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(secret, buf, &n, &err)
		tx.Data = buf.Bytes()

		// Set Sequence, Account, Signature
		pubKey, privKey = CreateKeys(secret)
		tx.SetSequence(0)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, "comit")

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())

		c, _ := net.Dial("tcp", am.ServerAddr)

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteMessage(ws.TextMessage, []byte(create_account_failure))
			continue
		}

		am.Info(Fmt("SUCCESS created account with pubKey %X\n", pubKey[:]))

		msg := Fmt(create_account_success, pubKey[:], privKey[:])

		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()

		return
	}
}

/*
func (am *ActionManager) RemoveAccount(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
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
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		c, _ := net.Dial("tcp", am.ServerAddr)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		var acc *types.Account
		accBytes := resQuery.Data
		err = wire.ReadBinaryBytes(accBytes, &acc)
		if err != nil {
			panic(err)
		}

		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)

		reqQuery = tmsp.ToRequestQuery([]byte{QUERY_CHAIN_ID})

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resQuery = res.GetQuery()

		if resQuery == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		chainID := resQuery.Log
		tx.SetSignature(privKey, chainID)

		// TxBytes in AppendTx request
		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
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
*/

func (am *ActionManager) Connect(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Data = []byte{CHECK_TX_CONNECT}

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
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
			am.Error(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Create connection to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		var acc *types.Account
		accBytes := res.GetQuery().Data
		err = wire.ReadBinaryBytes(accBytes, &acc)

		if err != nil {
			panic(err)
		}

		// Set sequence, account, signature
		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.ChainID)

		// TxBytes in CheckTx request
		txBytes := wire.BinaryBytes(tx)
		reqCheckTx := tmsp.ToRequestCheckTx(txBytes)

		err = am.WriteRequest(reqCheckTx, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resCheckTx := res.GetCheckTx()

		if resCheckTx == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		if resCheckTx.Code != tmsp.CodeType_OK {
			conn.WriteMessage(ws.TextMessage, []byte(connect_failure))
			continue
		}

		am.Info("SUCCESS connected to network")

		if !acc.IsAdmin() {
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
		panic(err)
	}

	// Create tx
	var tx types.Tx
	tx.Type = types.SubmitTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// Form
	var form *forms.Form

	for {

		// Form information
		_, issueBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		issue := string(issueBytes)

		_, locationBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		location := string(locationBytes)

		_, descriptionBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		description := string(descriptionBytes)

		_, mediaBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		_, extensionBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		extension := string(extensionBytes)

		_, anonymousBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		anonymous, _ := strconv.ParseBool(string(anonymousBytes))

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		n, err := hex.Decode(pubKey[:], pubKeyBytes)

		if err != nil || n != PUBKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// TODO: field validation
		if anonymous {
			form, err = forms.MakeAnonymousForm(
				// Text
				issue,
				location,
				description,
				// Media
				mediaBytes,
				extension)
		} else {
			pubKeyString := BytesToHexString(pubKey[:])
			form, err = forms.MakeForm(
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
			am.Error(err.Error())
			return //change
		}

		tx.Data = wire.BinaryBytes(*form)

		// PrivKey
		_, privKeyBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
			return
		}

		n, err = hex.Decode(privKey[:], privKeyBytes)

		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Connect to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		var acc *types.Account
		accBytes := resQuery.Data
		err = wire.ReadBinaryBytes(accBytes, &acc)
		if err != nil {
			panic(err)
		}

		// Set sequence, account, signature
		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.ChainID)

		// TxBytes in AppendTx
		txBytes := wire.BinaryBytes(tx)
		reqAppendTx := tmsp.ToRequestAppendTx(txBytes)

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteMessage(ws.TextMessage, []byte(submit_form_failure))
			continue
		}

		am.Info(Fmt("SUCCESS submitted form with ID %X\n", form.ID()))

		// Send response to ws

		msg := Fmt(submit_form_success, form.ID())
		conn.WriteMessage(ws.TextMessage, []byte(msg))

		// conn.Close()
		// return
	}
}

// Resolve form

func (am *ActionManager) ResolveForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
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
			am.Error(err.Error())
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
			am.Error(err.Error())
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
			am.Error(err.Error())
			return
		}

		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Connect to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		var acc *types.Account
		accBytes := res.GetQuery().Data
		err = wire.ReadBinaryBytes(accBytes, &acc)

		if err != nil {
			panic(err)
		}

		// Set sequence, account, signature
		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.ChainID)

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		am.Info(Fmt("SUCCESS resolved form with ID %X\n", formIDBytes))

		// Send response to ws
		msg := Fmt(resolve_form_success, formIDBytes)
		conn.WriteMessage(ws.TextMessage, []byte(msg))
		conn.Close()

		return
	}
}

func (am *ActionManager) FindForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	// Form
	var form forms.Form

	for {

		// Form ID
		_, hexBytes, err := conn.ReadMessage()

		if err != nil {
			am.Error(err.Error())
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

		// Connect to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		reqQuery := am.KeyQuery(key)
		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		if resQuery.Code != tmsp.CodeType_OK {
			msg := Fmt(find_form_failure, formIDBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}

		value := resQuery.Data
		err = wire.ReadBinaryBytes(value, &form)

		if err != nil {
			msg := Fmt(decode_form_failure, formIDBytes)
			conn.WriteMessage(ws.TextMessage, []byte(msg))
			panic(err)
		}

		am.Info(Fmt("SUCCESS found form with ID %X", formIDBytes))
		msg := (&form).Summary("find", 0)

		conn.WriteMessage(ws.TextMessage, []byte(msg))
	}
}

func (am *ActionManager) UpdateFeed(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	// Create Tx
	var tx types.Tx
	tx.Data = []byte{CHECK_TX_REGISTER}

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		// Issue
		_, issueBytes, err := conn.ReadMessage()
		if err != nil {
			am.Error(err.Error())
			return
		}

		issues := strings.Split(string(issueBytes), `,`)

		// PubKey
		_, pubKeyBytes, err := conn.ReadMessage()
		if err != nil {
			am.Error(err.Error())
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
			am.Error(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			conn.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
			continue
		}

		// Create connection to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(write_request_failure))
			continue
		}

		res := am.ReadResponse(c)

		var acc *types.Account
		accBytes := res.GetQuery().Data

		err = wire.ReadBinaryBytes(accBytes, &acc)
		if err != nil {
			panic(err)
		}

		// Set sequence, account, signature
		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)
		tx.SetSignature(privKey, am.ChainID)

		// TxBytes in CheckTx request
		txBytes := wire.BinaryBytes(tx)
		reqCheckTx := tmsp.ToRequestCheckTx(txBytes)

		am.WriteRequest(reqCheckTx, c)

		// --------------------------- //

		am.Info("Updating feed...")

		cli := NewClient(c, conn)

		// Read updates from TMSP server conn
		// Write updates to websocket...
		done := make(chan struct{})
		go cli.WriteRoutine(issues, done)
		go cli.ReadRoutine()

		<-done
		break
	}
}

/*
func (am *ActionManager) SearchForms(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
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
			minutestr := string(forms.XOR(key, issue))
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

		var form forms.Form

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
		panic(err)
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
		panic(err)
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

// Create Admin
func (am *ActionManager) CreateAdmin(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
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
		accKey := state.AccountKey(addr)

		buf = new(bytes.Buffer)
		wire.WriteByteSlice(accKey, buf, &n, &err)
		key := buf.Bytes()
		reqQuery := am.KeyQuery(key)
		res, err := am.WriteRequest(reqQuery)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		var acc types.Account
		accBytes := res.GetQuery().Data
		err = wire.ReadBinaryBytes(accBytes, &acc)
		if err != nil {
			panic(err)
		}

		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)

		reqQuery = tmsp.ToRequestQuery([]byte{QUERY_CHAIN_ID})
		res, err = am.WriteRequest(reqQuery)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		chainID := res.GetQuery().Log
		tx.SetSignature(privKey, chainID)

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())
		res, err = am.WriteRequest(reqAppendTx)

		resAppendTx := res.GetAppendTx()

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteMessage(ws.TextMessage, []byte(create_admin_failure))
			continue
		}

		pubKeyBytes, _, err = wire.GetByteSlice(resAppendTx.Data)
		if err != nil {
			panic(err)
		}
		privKeyBytes, _, err = wire.GetByteSlice(resAppendTx.Data)
		if err != nil {
			panic(err)
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
		panic(err)
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

		// Set Sequence, Account, Signature
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)

		buf := new(bytes.Buffer)
		wire.WriteByteSlice(accKey, buf, &n, &err)
		key := buf.Bytes()
		reqQuery := am.KeyQuery(key)
		res, err := am.WriteRequest(reqQuery)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		var acc types.Account
		accBytes := res.GetQuery().Data
		err = wire.ReadBinaryBytes(accBytes, &acc)
		if err != nil {
			panic(err)
		}

		tx.SetSequence(acc.Sequence)
		tx.SetAccount(pubKey)

		reqQuery = tmsp.ToRequestQuery([]byte{QUERY_CHAIN_ID})
		res, err = am.WriteRequest(reqQuery)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		chainID := res.GetQuery().Log
		tx.SetSignature(privKey, chainID)

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())
		res, err = am.WriteRequest(reqAppendTx)

		if err != nil {
			conn.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx.Code != tmsp.CodeType_OK {
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
*/
