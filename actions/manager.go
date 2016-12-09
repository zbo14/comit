package actions

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	ws "github.com/gorilla/websocket"
	"github.com/tendermint/go-crypto"
	wire "github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/forms"
	"github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	// "io"
	// "compress/flate"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	MAX_MEMORY int64 = 1000000000000

	PUBKEY_LENGTH  = 32
	PRIVKEY_LENGTH = 64
	MOMENT_LENGTH  = 32
	FORM_ID_LENGTH = 16

	QueryChainID byte = 0
	QuerySize    byte = 1
	QueryKey     byte = 2
	QueryIndex   byte = 3
	QueryIssues  byte = 4
	QuerySearch  byte = 5
)

var Fmt = fmt.Sprintf

type ActionManager struct {
	ServerAddr string
	ChainID    string
	Issues     []string

	ConnsMtx sync.Mutex
	Conns    map[string]net.Conn

	// FormsMtx sync.Mutex
	// Forms    map[string]chan *forms.Form

	types.Logger
}

func CreateActionManager(serverAddr string) *ActionManager {

	am := &ActionManager{
		ServerAddr: serverAddr,
		Conns:      make(map[string]net.Conn),
		Logger:     types.NewLogger("action-manager"),
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

func (am *ActionManager) SearchQuery(search []byte) *tmsp.Request {
	query := make([]byte, wire.ByteSliceSize(search)+1)
	buf := query
	buf[0] = QuerySearch
	buf = buf[1:]
	wire.PutByteSlice(buf, search)
	req := tmsp.ToRequestQuery(query)
	return req
}

// Read, Write

func (am *ActionManager) ReadResponse(ws net.Conn) *tmsp.Response {

	bufReader := bufio.NewReader(ws)
	res := &tmsp.Response{}
	err := tmsp.ReadMessage(bufReader, res)
	if err != nil {
		return nil
	}
	return res
}

func (am *ActionManager) WriteRequest(req *tmsp.Request, ws net.Conn) error {

	bufWriter := bufio.NewWriter(ws)
	err := tmsp.WriteMessage(req, bufWriter)
	if err != nil {
		return err
	}
	flush := tmsp.ToRequestFlush()
	tmsp.WriteMessage(flush, bufWriter)
	bufWriter.Flush()
	return nil
}

// ChainID

func (am *ActionManager) GetChainID() {

	reqQuery := tmsp.ToRequestQuery([]byte{QueryChainID})
	conn, _ := net.Dial("tcp", am.ServerAddr)

	err := am.WriteRequest(reqQuery, conn)

	if err != nil {
		panic(err)
	}

	res := am.ReadResponse(conn)

	if res == nil {
		panic(err)
	}

	resQuery := res.GetQuery()

	if resQuery == nil {
		panic(err)
	}

	am.ChainID = resQuery.Log
}

// Issues

func (am *ActionManager) GetIssues() {

	reqQuery := tmsp.ToRequestQuery([]byte{QueryIssues})
	conn, _ := net.Dial("tcp", am.ServerAddr)

	err := am.WriteRequest(reqQuery, conn)

	if err != nil {
		panic(err)
	}

	res := am.ReadResponse(conn)

	if res == nil {
		panic(err)
	}

	resQuery := res.GetQuery()

	if resQuery == nil {
		panic(err)
	}

	var issues []string
	err = wire.ReadBinaryBytes(resQuery.Data, &issues)
	if err != nil {
		panic(err)
	}

	am.Issues = issues
}

func (am *ActionManager) SendIssues(w http.ResponseWriter, req *http.Request) {

	var buf bytes.Buffer
	buf.WriteString(Fmt(select_option, "", "select issue"))

	for _, issue := range am.Issues {
		buf.WriteString(Fmt(select_option, issue, issue))
	}

	data := buf.Bytes()

	// Status OK
	w.WriteHeader(200)

	w.Write(data)
}

// Create Account
func (am *ActionManager) CreateAccount(w http.ResponseWriter, req *http.Request) {

	// Request data
	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(w, HttpRequestFailure, http.StatusBadRequest)
		return
	}

	v, _ := url.ParseQuery(string(data))

	secret := v.Get("secret")

	// Create Tx
	var tx types.Tx
	tx.Type = types.CreateAccountTx
	tx.Data = []byte(secret)

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// Set Sequence, Account, Signature
	pubKey, privKey = CreateKeys([]byte(secret))
	tx.SetSequence(0)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.ChainID)

	// TxBytes in AppendTx request
	txBytes := wire.BinaryBytes(tx)
	reqAppendTx := tmsp.ToRequestAppendTx(txBytes)

	// Connect to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	err = am.WriteRequest(reqAppendTx, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resAppendTx := am.ReadResponse(conn).GetAppendTx()

	// Status OK
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	if resAppendTx.Code != tmsp.CodeType_OK {
		enc.Encode(CreateAccount{Result: "failure"})
		return
	}

	pubKeystr := BytesToHexString(pubKey[:])
	privKeystr := BytesToHexString(privKey[:])

	enc.Encode(CreateAccount{"success", pubKeystr, privKeystr})
}

func (am *ActionManager) RemoveAccount(w http.ResponseWriter, req *http.Request) {

	// Request data
	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(w, HttpRequestFailure, http.StatusBadRequest)
		return
	}

	v, _ := url.ParseQuery(string(data))

	pubKeystr := v.Get("public-key")
	privKeystr := v.Get("private-key")

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// PubKey
	pubKeyBytes, err := hex.DecodeString(pubKeystr)

	if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
		http.Error(w, InvalidPublicKey, http.StatusUnauthorized)
		return
	}

	copy(pubKey[:], pubKeyBytes)

	// PrivKey
	privKeyBytes, err := hex.DecodeString(privKeystr)

	if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
		http.Error(w, InvalidPrivateKey, http.StatusUnauthorized)
		return
	}

	copy(privKey[:], privKeyBytes)

	// Create Tx
	var tx types.Tx
	tx.Type = types.RemoveAccountTx

	// Connect to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	// Query sequence
	addr := pubKey.Address()
	accKey := state.AccountKey(addr)
	reqQuery := am.KeyQuery(accKey)

	err = am.WriteRequest(reqQuery, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resQuery := am.ReadResponse(conn).GetQuery()

	var acc *types.Account
	accBytes := resQuery.Data

	wire.ReadBinaryBytes(accBytes, &acc)

	// Set sequence, account, signature
	tx.SetSequence(acc.Sequence)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.ChainID)

	// TxBytes in AppendTx request
	txBytes := wire.BinaryBytes(tx)
	reqAppendTx := tmsp.ToRequestAppendTx(txBytes)

	err = am.WriteRequest(reqAppendTx, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resAppendTx := am.ReadResponse(conn).GetAppendTx()

	// Status OK
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	if resAppendTx.Code != tmsp.CodeType_OK {
		enc.Encode(RemoveAccount{"failure"})
		return
	}

	enc.Encode(RemoveAccount{"success"})
}

func (am *ActionManager) Login(w http.ResponseWriter, req *http.Request) {

	// Request data
	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(w, HttpRequestFailure, http.StatusBadRequest)
		return
	}

	v, _ := url.ParseQuery(string(data))

	pubKeystr := v.Get("public-key")
	privKeystr := v.Get("private-key")

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// PubKey
	pubKeyBytes, err := hex.DecodeString(pubKeystr)

	if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
		http.Error(w, InvalidPublicKey, http.StatusUnauthorized)
		return
	}

	copy(pubKey[:], pubKeyBytes)

	// PrivKey
	privKeyBytes, err := hex.DecodeString(privKeystr)

	if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
		http.Error(w, InvalidPrivateKey, http.StatusUnauthorized)
		return
	}

	copy(privKey[:], privKeyBytes)

	// Create connection to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	// Create Tx
	var tx types.Tx
	tx.Type = types.ConnectTx

	// Query sequence
	addr := pubKey.Address()
	accKey := state.AccountKey(addr)
	reqQuery := am.KeyQuery(accKey)

	err = am.WriteRequest(reqQuery, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	res := am.ReadResponse(conn)

	var acc *types.Account
	accBytes := res.GetQuery().Data

	wire.ReadBinaryBytes(accBytes, &acc)

	// Set sequence, account, signature
	tx.SetSequence(acc.Sequence)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.ChainID)

	// TxBytes in CheckTx request
	txBytes := wire.BinaryBytes(tx)
	reqCheckTx := tmsp.ToRequestCheckTx(txBytes)

	err = am.WriteRequest(reqCheckTx, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resCheckTx := am.ReadResponse(conn).GetCheckTx()

	// Status OK
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	if resCheckTx.Code != tmsp.CodeType_OK {
		enc.Encode(Connect{Result: "failure"})
		return
	}

	if acc.IsAdmin() {
		enc.Encode(Connect{"success", "admin"})
		return
	}

	enc.Encode(Connect{"success", "constituent"})
}

func (am *ActionManager) SubmitForm(w http.ResponseWriter, req *http.Request) {

	mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))

	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	mr := multipart.NewReader(req.Body, params["boundary"])

	f, err := mr.ReadForm(MAX_MEMORY)

	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// Form // TODO: field validation

	form := forms.Form{}

	// Info

	form.SubmittedAt = time.Now().Local().String()
	form.Issue = f.Value["issue"][0]
	form.Location = f.Value["location"][0]
	form.Description = f.Value["description"][0]

	// Keys

	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	pubKeystr := f.Value["public-key"][0]
	privKeystr := f.Value["private-key"][0]

	pubKeyBytes, err := hex.DecodeString(pubKeystr)

	if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
		http.Error(w, InvalidPublicKey, http.StatusUnauthorized)
		return
	}

	copy(pubKey[:], pubKeyBytes)

	privKeyBytes, err := hex.DecodeString(privKeystr)

	if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
		http.Error(w, InvalidPrivateKey, http.StatusUnauthorized)
		return
	}

	copy(privKey[:], privKeyBytes)

	form.Submitter = pubKeystr

	// Media

	media := f.File["media"][0]

	form.ContentType = media.Header.Get("Content-Type")

	file, err := media.Open()

	if err != nil {
		panic(err)
	}

	form.Data, err = ioutil.ReadAll(file)

	if err != nil {
		panic(err)
	}

	// Create tx
	var tx types.Tx
	tx.Type = types.SubmitTx
	tx.Data = wire.BinaryBytes(form)

	// Query sequence
	addr := pubKey.Address()
	accKey := state.AccountKey(addr)
	reqQuery := am.KeyQuery(accKey)

	// Connect to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	err = am.WriteRequest(reqQuery, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resQuery := am.ReadResponse(conn).GetQuery()

	var acc *types.Account
	accBytes := resQuery.Data

	wire.ReadBinaryBytes(accBytes, &acc)

	// Set sequence, account, signature
	tx.SetSequence(acc.Sequence)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.ChainID)

	// TxBytes in AppendTx
	txBytes := wire.BinaryBytes(tx)
	reqAppendTx := tmsp.ToRequestAppendTx(txBytes)

	err = am.WriteRequest(reqAppendTx, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resAppendTx := am.ReadResponse(conn).GetAppendTx()

	// Status OK
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	if resAppendTx.Code != tmsp.CodeType_OK {
		enc.Encode(SubmitForm{Result: "failure"})
		return
	}

	formID := BytesToHexString(form.ID())

	enc.Encode(SubmitForm{"success", formID})
}

/*
// Resolve form

func (am *ActionManager) ResolveForm(w http.ResponseWriter, req *http.Request) {

	// Websocket wsection
	ws, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer ws.Close()

	// Connect to TMSP server
	conn,_ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

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

		err = ws.ReadJSON(v)

		if err != nil {
			panic(err)
		}

		// Form ID
		formID, err := hex.DecodeString(v.FormID)

		if err != nil || len(formID) != FORM_ID_LENGTH {
			ws.WriteJSON(InvalidFormID)
			continue
		}

		// PubKey
		pubKeyBytes, err := hex.DecodeString(v.PubKeystr)

		if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
			ws.WriteJSON(InvalidPublicKey)
			continue
		}

		copy(pubKey[:], pubKeyBytes)

		// PrivKey
		privKeyBytes, err := hex.DecodeString(v.PrivKeystr)

		if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
			ws.WriteJSON(InvalidPrivateKey)
			continue
		}

		copy(privKey[:], privKeyBytes)

		// Query sequence
		addr := pubKey.Address()
		accKey := state.AccountKey(addr)
		reqQuery := am.KeyQuery(accKey)

		err = am.WriteRequest(reqQuery, conn)

		if err != nil {
			ws.WriteJSON(WriteRequestError)
			continue
		}

		res := am.ReadResponse(conn)

		if res == nil {
			ws.WriteJSON(ReadResponseError)
			continue
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			ws.WriteJSON(ReadResponseError)
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

		err = am.WriteRequest(reqAppendTx, conn)

		if err != nil {
			ws.WriteJSON(WriteRequestError)
			continue
		}

		res = am.ReadResponse(conn)

		if res == nil {
			ws.WriteJSON(ReadResponseError)
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx == nil {
			ws.WriteJSON(ReadResponseError)
			continue
		}

		if resAppendTx.Code != tmsp.CodeType_OK {
			ws.WriteJSON(ResolveFormFailure)
			continue
		}

		ws.WriteJSON(ResolveFormSuccess)

		return
	}
}

*/

func (am *ActionManager) FindForm(w http.ResponseWriter, req *http.Request) {

	// Request data
	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(w, HttpRequestFailure, http.StatusBadRequest)
		return
	}

	v, _ := url.ParseQuery(string(data))

	formIDstr := v.Get("form-ID")
	// pubKeystr := v.Get("public-key")

	am.Info(formIDstr)

	formID, err := hex.DecodeString(formIDstr)
	if err != nil {
		am.Error(err.Error())
		http.Error(w, InvalidFormID, http.StatusNotFound)
		return
	}

	if len(formID) != FORM_ID_LENGTH {
		am.Info("Form ID", "length", len(formID))
		http.Error(w, InvalidFormID, http.StatusNotFound)
		return
	}

	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(formID, buf, &n, &err)
	key := buf.Bytes()

	reqQuery := am.KeyQuery(key)

	// Connect to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	err = am.WriteRequest(reqQuery, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	resQuery := am.ReadResponse(conn).GetQuery()

	// Status OK // Form text
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	if resQuery.Code != tmsp.CodeType_OK {
		enc.Encode(FindForm{Result: "failure"})
		return
	}

	// Form
	form := &forms.Form{}

	value := resQuery.Data
	wire.ReadBinaryBytes(value, form)

	enc.Encode(FindForm{"success", form})
}

func (am *ActionManager) UpdateFeed(w http.ResponseWriter, req *http.Request) {

	// Request data
	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(w, HttpRequestFailure, http.StatusBadRequest)
		return
	}

	v, _ := url.ParseQuery(string(data))

	issue := v.Get("issue")
	pubKeystr := v.Get("public-key")
	privKeystr := v.Get("private-key")

	// Keys
	var pubKey crypto.PubKeyEd25519
	var privKey crypto.PrivKeyEd25519

	// PubKey
	pubKeyBytes, err := hex.DecodeString(pubKeystr)

	if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
		http.Error(w, InvalidPublicKey, http.StatusUnauthorized)
		return
	}

	copy(pubKey[:], pubKeyBytes)

	// PrivKey
	privKeyBytes, err := hex.DecodeString(privKeystr)

	if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
		http.Error(w, InvalidPrivateKey, http.StatusUnauthorized)
		return
	}

	copy(privKey[:], privKeyBytes)

	// Create connection to TMSP server
	conn, _ := net.Dial("tcp", am.ServerAddr)

	// Do not close conn!

	// Query sequence
	addr := pubKey.Address()
	accKey := state.AccountKey(addr)
	reqQuery := am.KeyQuery(accKey)

	err = am.WriteRequest(reqQuery, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	res := am.ReadResponse(conn)

	var acc *types.Account
	accBytes := res.GetQuery().Data

	wire.ReadBinaryBytes(accBytes, &acc)

	// Create Tx
	var tx types.Tx
	tx.Type = types.UpdateTx

	// Set sequence, account, signature
	tx.SetSequence(acc.Sequence)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.ChainID)

	// TxBytes in CheckTx request
	txBytes := wire.BinaryBytes(tx)
	reqCheckTx := tmsp.ToRequestCheckTx(txBytes)

	err = am.WriteRequest(reqCheckTx, conn)

	if err != nil {
		http.Error(w, TmspRequestFailure, http.StatusInternalServerError)
		return
	}

	// Save conn
	am.ConnsMtx.Lock()
	am.Conns[pubKeystr] = conn
	am.ConnsMtx.Unlock()

	// Status OK
	w.WriteHeader(200)

	// Encode JSON
	enc := json.NewEncoder(w)

	enc.Encode(struct {
		Issue     string `json:"issue"`
		PubKeystr string `json:"public-key"`
	}{issue, pubKeystr})

}

func (am *ActionManager) Updates(w http.ResponseWriter, req *http.Request) {

	// Websocket
	ws, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer ws.Close()

	v := &struct {
		Issue     string `json:"issue"`
		PubKeystr string `json:"public-key"`
	}{}

	// Read from websocket
	err = ws.ReadJSON(v)

	if err != nil {
		panic(err) //for now
	}

	log.Printf("%v\n", v)

	// Get conn to TMSP server

	am.ConnsMtx.Lock()
	conn, ok := am.Conns[v.PubKeystr]
	am.ConnsMtx.Unlock()

	defer conn.Close()

	if !ok {
		panic("Could not find conn") //for now
	}

	am.Info("Updating feed...")

	cli := NewClient(conn, ws)

	done := make(chan struct{})

	// Read updates from TMSP server connection
	// Write updates to websocket
	go cli.WriteRoutine(v.Issue, done)
	go cli.ReadRoutine()

	<-done

	am.Info("Done")
}

/*
func (am *ActionManager) SearchForms(w http.ResponseWriter, req *http.Request) {

	// Websocket wsection
	ws, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer ws.Close()

	// Create connection to TMSP server
	conn,_ := net.Dial("tcp", am.ServerAddr)
	defer conn.Close()

	for {

		// TODO: add location

		v := &struct {
			After  string `json:"after"`
			Before string `json:"before"`
			Issue  string `json:"issue"`
		}{}

		err = ws.ReadJSON(v)

		if err != nil {
			panic(err)
		}

		// Search
		s := struct {
			AfterTime  time.Time
			BeforeTime time.Time
			Issue      string
		}{}

		s.AfterTime = ParseMomentString(v.After)
		s.BeforeTime = ParseMomentString(v.Before)
		s.Issue = v.Issue

		log.Println(v.After)
		log.Printf("%v\n", s.AfterTime)

		search := wire.BinaryBytes(s)
		reqQuery := am.SearchQuery(search)

		err = am.WriteRequest(reqQuery, conn)

		if err != nil {
			panic(err)
		}

		res := am.ReadResponse(conn)

		if res == nil {
			panic(err)
		}

		resQuery := res.GetQuery()

		if resQuery == nil {
			panic(err)
		}

		am.Info("Search finished")

		var datas [][]byte
		wire.ReadBinaryBytes(resQuery.Data, &datas)

		log.Printf("%v\n", datas)
	}
}

/*
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
				ws.WriteMessage(ws.TextMessage, []byte(msg))
				count++
			} else {
				break
			}
		}


NO MESSAGES FOR NOW

func (am *ActionManager) CheckMessages(w http.ResponseWriter, req *http.Request) {

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	for {

		var pubKey crypto.PubKeyEd25519
		_, pubKeyBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			return
		}

		log.Println("Getting messages...")

		// Create client
		cli := NewClient(ws, pubKey)

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

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	for {

		message := NewMessage()

		_, recipientBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err := hex.Decode(message.recipient[:], recipientBytes)
		if err != nil || n != PUBKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		_, contentBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		message.content = contentBytes

		_, senderBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(message.sender[:], senderBytes)
		if err != nil || n != PUBKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		err = am.sendMessage(message)
		if err != nil {
			ws.WriteMessage(ws.TextMessage, []byte("Could not send message"))
			ws.Close()
			return
		}

		ws.WriteMessage(ws.TextMessage, []byte("Message sent!"))
		// ws.Close()
		// return
	}
}

NO ADMIN FOR NOW

// Create Admin
func (am *ActionManager) CreateAdmin(w http.ResponseWriter, req *http.Request) {

	ws, err := upgrader.Upgrade(w, req, nil)
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
		_, secret, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteByteSlice(secret, buf, &n, &err)
		tx.Data = buf.Bytes()

		// PubKey
		_, pubKeyBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}
		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
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
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
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
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
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
			ws.WriteMessage(ws.TextMessage, []byte(create_admin_failure))
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
		ws.WriteMessage(ws.TextMessage, []byte(msg))
		ws.Close()
		return
	}
}

// Remove admin

func (am *ActionManager) RemoveAdmin(w http.ResponseWriter, req *http.Request) {

	ws, err := upgrader.Upgrade(w, req, nil)
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
		_, pubKeyBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		n, err := hex.Decode(pubKey[:], pubKeyBytes)
		if err != nil || n != PUBKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_public_key))
			continue
		}

		// PrivKey
		_, privKeyBytes, err := ws.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		n, err = hex.Decode(privKey[:], privKeyBytes)
		if err != nil || n != PRIVKEY_LENGTH {
			ws.WriteMessage(ws.TextMessage, []byte(invalid_private_key))
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
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
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
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
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
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		resAppendTx := res.GetAppendTx()

		if resAppendTx.Code != tmsp.CodeType_OK {
			msg := Fmt(remove_admin_failure, pubKeyBytes)
			ws.WriteMessage(ws.TextMessage, []byte(msg))
			continue
		}

		log.Printf("SUCCESS removed admin with pubKey %X\n", pubKeyBytes)
		msg := Fmt(remove_admin_success, pubKeyBytes)
		ws.WriteMessage(ws.TextMessage, []byte(msg))
		ws.Close()
		return
	}
}
*/
