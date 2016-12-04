package actions

import (
	"bufio"
	"bytes"
	"encoding/hex"
	// "encoding/json"
	ws "github.com/gorilla/websocket"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/forms"
	"github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"log"
	"net"
	"net/http"
	// "strings"
	//"time"
)

const (
	PUBKEY_LENGTH  = 32
	PRIVKEY_LENGTH = 64
	MOMENT_LENGTH  = 32
	FORM_ID_LENGTH = 16

	QueryChainID byte = 0
	QuerySize    byte = 1
	QueryKey     byte = 2
	QueryIndex   byte = 3
	QueryIssues  byte = 4
)

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
	buf[0] = QueryKey
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	req := tmsp.ToRequestQuery(query)
	return req
}

func (am *ActionManager) IndexQuery(i int) *tmsp.Request {
	query := make([]byte, 100)
	buf := query
	buf[0] = QueryIndex
	buf = buf[1:]
	n, err := wire.PutVarint(buf, i)
	if err != nil {
		return nil
	}
	query = query[:n+1]
	req := tmsp.ToRequestQuery(query)
	return req
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

// Get chainID, issues

func (am *ActionManager) GetChainID() {

	for {

		reqQuery := tmsp.ToRequestQuery([]byte{QueryChainID})
		c, _ := net.Dial("tcp", am.ServerAddr)

		err := am.WriteRequest(reqQuery, c)

		if err != nil {
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			continue
		}

		am.ChainID = resQuery.Log
		return
	}
}

func (am *ActionManager) GetIssues() {

	for {

		reqQuery := tmsp.ToRequestQuery([]byte{QueryIssues})
		c, _ := net.Dial("tcp", am.ServerAddr)

		err := am.WriteRequest(reqQuery, c)

		if err != nil {
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
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

	defer conn.Close()

	var buf bytes.Buffer
	buf.WriteString(Fmt(select_option, "", "select issue"))

	for _, issue := range am.Issues {
		buf.WriteString(Fmt(select_option, issue, issue))
	}

	conn.WriteMessage(ws.TextMessage, buf.Bytes())

	return
}

// Create Account
func (am *ActionManager) CreateAccount(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	// Connect to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

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
		tx.SetSignature(privKey, am.ChainID)

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(c)

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteJSON(CreateAccountFailure)
			continue
		}

		pubKeystr := BytesToHexString(pubKey[:])
		privKeystr := BytesToHexString(privKey[:])

		err = conn.WriteJSON(CreateAccount{"success", pubKeystr, privKeystr})

		log.Println(err)

		return
	}
}

func (am *ActionManager) RemoveAccount(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	// Connect to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

	// Create Tx
	var tx types.Tx
	tx.Type = types.RemoveAccountTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		v := &struct {
			PubKeystr  string `json:"public-key"`
			PrivKeystr string `json:"private-key"`
		}{}

		err = conn.ReadJSON(v)

		if err != nil {
			am.Error(err.Error())
			return
		}

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKey[:]) != PUBKEY_LENGTH {
			conn.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKey[:]) != PRIVKEY_LENGTH {
			conn.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)

		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteJSON(ReadResponseError)
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

		// TxBytes in AppendTx request
		txBytes := wire.BinaryBytes(tx)
		reqAppendTx := tmsp.ToRequestAppendTx(txBytes)

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteJSON(RemoveAccountFailure)
			continue
		}

		conn.WriteJSON(RemoveAccountSuccess)

		return
	}
}

func (am *ActionManager) Connect(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	// Create Tx
	var tx types.Tx
	tx.Type = types.ConnectTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		v := &struct {
			PubKeystr  string `json:"public-key"`
			PrivKeystr string `json:"private-key"`
		}{}

		err = conn.ReadJSON(v)

		if err != nil {
			am.Error(err.Error())
			return
		}

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
			conn.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
			conn.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// Create connection to TMSP server
		c, _ := net.Dial("tcp", am.ServerAddr)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
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
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resCheckTx := res.GetCheckTx()

		if resCheckTx == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resCheckTx.Code != tmsp.CodeType_OK {
			conn.WriteJSON(ConnectFailure)
			continue
		}

		if !acc.IsAdmin() {
			conn.WriteJSON(ConnectConstituent)
			return
		}

		conn.WriteJSON(ConnectAdmin)
		return
	}
}

func (am *ActionManager) SubmitForm(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	// defer conn.Close()
	// don't close if user wants to submit
	// multiple forms consecutively

	// Connect to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

	// Create tx
	var tx types.Tx
	tx.Type = types.SubmitTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// Form
	var form *forms.Form

	for {

		_, media, err := conn.ReadMessage()

		if err != nil {
			panic(err)
		}

		v := &struct {
			Issue       string `json:"issue"`
			Location    string `json:"location"`
			Description string `json:"description"`
			Extension   string `json:"extension"`
			Anonymous   bool   `json:"anonymous"`
			PubKeystr   string `json:"public-key"`
			PrivKeystr  string `json:"private-key"`
		}{}

		err = conn.ReadJSON(v)

		if err != nil {
			panic(err)
		}

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
			conn.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
			conn.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// TODO: field validation
		if v.Anonymous {
			form, err = forms.MakeAnonymousForm(
				// Text
				v.Issue,
				v.Location,
				v.Description,
				// Media
				media,
				v.Extension)
		} else {
			form, err = forms.MakeForm(
				// Text
				v.Issue,
				v.Location,
				v.Description,
				v.PubKeystr,
				// Media
				media,
				v.Extension)
		}

		if err != nil {
			am.Error(err.Error())
			return
		}

		tx.Data = wire.BinaryBytes(*form)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteJSON(ReadResponseError)
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
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteJSON(SubmitFormFailure)
			continue
		}

		formID := BytesToHexString(form.ID())

		conn.WriteJSON(SubmitForm{"success", formID})

		// return
		// dont' return if user wants to submit
		// multiple forms consecutively
	}
}

// Resolve form

func (am *ActionManager) ResolveForm(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	// Connect to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

	// Create Tx
	var tx types.Tx
	tx.Type = types.ResolveTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		v := &struct {
			FormID     string `json:"form-ID"`
			PubKeystr  string `json:"public-key"`
			PrivKeystr string `json:"private-key"`
		}{}

		err = conn.ReadJSON(v)

		if err != nil {
			panic(err)
		}

		// Form ID
		formID, err := hex.DecodeString(v.FormID)

		if err != nil || len(formID) != FORM_ID_LENGTH {
			conn.WriteJSON(InvalidFormID)
			continue
		}

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
			conn.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
			conn.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteJSON(ReadResponseError)
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
		buf, n := new(bytes.Buffer), int(0)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())

		err = am.WriteRequest(reqAppendTx, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res = am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			conn.WriteJSON(ResolveFormFailure)
			continue
		}

		conn.WriteJSON(ResolveFormSuccess)

		return
	}
}

func (am *ActionManager) FindForm(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	// defer conn.Close
	// no close if user wants to query
	// multiple forms consecutively

	// Connect to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

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
			conn.WriteJSON(InvalidFormID)
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(formIDBytes, buf, &n, &err)
		key := buf.Bytes()

		reqQuery := am.KeyQuery(key)
		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(c)

		if res == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			conn.WriteJSON(ReadResponseError)
			continue
		}

		if resQuery.Code != tmsp.CodeType_OK {
			conn.WriteJSON(FindFormFailure)
			continue
		}

		value := resQuery.Data
		wire.ReadBinaryBytes(value, &form)

		conn.WriteJSON(FindForm{"success", form})
	}
}

func (am *ActionManager) UpdateFeed(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	// Create connection to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

	// Create Tx
	var tx types.Tx
	tx.Type = types.UpdateTx

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	for {

		v := &struct {
			Issues     []string `json:"issues"`
			PubKeystr  string   `json:"public-key"`
			PrivKeystr string   `json:"private-key"`
		}{}

		err = conn.ReadJSON(v)

		if err != nil {
			panic(err)
		}

		log.Printf("%v\n", v)

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
			conn.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
			conn.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			conn.WriteJSON(WriteRequestError)
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

		// --------------------------------------- //

		am.Info("Updating feed...")

		cli := NewClient(c, conn)

		done := make(chan struct{})

		// Read updates from TMSP server conn
		// Write updates to websocket...
		go cli.WriteRoutine(v.Issues, done)
		go cli.ReadRoutine()

		<-done
		break
	}
}

/*

NO SEARCH FORMS FOR NOW

func (am *ActionManager) SearchForms(w http.ResponseWriter, req *http.Request) {

	// Websocket connection
	conn, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	// Create connection to TMSP server
	c, _ := net.Dial("tcp", am.ServerAddr)
	defer c.Close()

	// Times
	var afterTime time.Time
	var beforeTime time.Time

	count := 1

	for {

		afterTime = time.Time{}
		beforeTime = time.Time{}

		_, afterBytes, err := conn.ReadMessage()
		if err != nil {
			am.Error(err.Error())
			return
		}

		after := string(afterBytes)
		if len(after) >= MOMENT_LENGTH {
			afterTime = ParseMomentString(after)
		}

		_, beforeBytes, err := conn.ReadMessage()
		if err != nil {
			am.Error(err.Error())
			return
		}

		before := string(beforeBytes)
		if len(before) >= MOMENT_LENGTH {
			beforeTime = ParseMomentString(before)
		}

		var filters []string

		_, issueBytes, err := conn.ReadMessage()
		if err != nil {
			am.Error(err.Error())
			return
		}

		issue := string(issueBytes)
		if len(issue) > 0 {
			filters = append(filters, issue)
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
		}

		reqQuery := am.FilterQuery(filters)
		err = am.WriteRequest(reqQuery, c)

		if err != nil {
			//
		}

		res := am.ReadResponse(c)

		if res == nil {
			//
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			//
		}

		log.Println("Search finished")
	}
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


NO MESSAGES FOR NOW

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

NO ADMIN FOR NOW

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

		reqQuery = tmsp.ToRequestQuery([]byte{QueryChainID})
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

		reqQuery = tmsp.ToRequestQuery([]byte{QueryChainID})
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
