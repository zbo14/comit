package manager

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	wire "github.com/tendermint/go-wire"
	tndr "github.com/tendermint/tendermint/types"
	"github.com/zballs/comit/state"
	. "github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"io/ioutil"
	"net/http"
	"time"
)

type Manager struct {
	proxy  *Proxy
	logger Logger

	acc     *PrivAccount
	chainID string
	issues  []string

	latestHeight int
	blocks       chan *tndr.Block
	headers      chan *tndr.Header
}

func CreateManager(remote, endpoint string) *Manager {
	return &Manager{
		proxy:   NewProxy(remote, endpoint),
		logger:  NewLogger("action-manager"),
		blocks:  make(chan *tndr.Block),
		chainID: "comit",
	}
}

func ManagerRespond(w http.ResponseWriter, res interface{}) {
	data, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (m *Manager) AddRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/issues", m.Issues)
	mux.HandleFunc("/login", m.Login)
	mux.HandleFunc("/create_account", m.CreateAccount)
	mux.HandleFunc("/remove_account", m.RemoveAccount)
	mux.HandleFunc("/submit_form", m.SubmitForm)
	mux.HandleFunc("/find_form", m.FindForm)
	mux.HandleFunc("/updates", m.Updates)
}

// ChainID

func (m *Manager) GetChainID(w http.ResponseWriter, req *http.Request) {

	query := EmptyQuery(QueryChainID)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageChainID(err, nil))
		return
	}

	tmResult := result.Result

	if tmResult.Code != 0 {
		data, _, err := wire.GetByteSlice(tmResult.Data)
		if err != nil {
			panic(err)
		}
		m.chainID = string(data)
	}

	ManagerRespond(w, MessageChainID(nil, result))
}

// Issues

func (m *Manager) GetIssues(w http.ResponseWriter, req *http.Request) {

	query := EmptyQuery(QueryIssues)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageIssues(err, nil, nil))
		return
	}

	tmResult := result.Result

	if tmResult.Code != 0 {
		ManagerRespond(w, MessageIssues(nil, nil, result))
		return
	}

	var issues []string
	wire.ReadBinaryBytes(tmResult.Data, &issues)
	m.issues = issues

	ManagerRespond(w, MessageIssues(nil, m.issues, result))
}

func (m *Manager) Issues(w http.ResponseWriter, req *http.Request) {

	if len(m.issues) == 0 {
		m.GetIssues(w, req)
		return
	}

	ManagerRespond(w, MessageIssues(nil, m.issues, nil))
}

func (m *Manager) Login(w http.ResponseWriter, req *http.Request) {

	// Get request data
	vals, err := UrlValues(req)

	if err != nil {
		http.Error(w, "Failed to read request data", http.StatusBadRequest)
		return
	}

	pubKeystr := vals.Get("pub_key")
	privKeystr := vals.Get("priv_key")

	// PubKey
	pubKey, err := PubKeyfromHexstr(pubKeystr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// PrivKey
	privKey, err := PrivKeyfromHexstr(privKeystr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Verify keypair
	sig := privKey.Sign([]byte(m.chainID))
	verified := pubKey.VerifyBytes([]byte(m.chainID), sig)

	if !verified {
		http.Error(w, "Invalid keypair", http.StatusUnauthorized)
		return
	}

	// Query account
	acckey := state.AccountKey(pubKey.Address())
	query := KeyQuery(acckey)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageLogin(err, nil))
		return
	}

	tmResult := result.Result

	var acc *Account
	err = wire.ReadBinaryBytes(tmResult.Data, &acc)

	if err != nil {
		panic(err)
	}

	m.acc = NewPrivAccount(acc, privKey)

	ManagerRespond(w, MessageLogin(nil, result))

	// Start proxy ws
	err = m.proxy.StartWS()
	if err != nil {
		panic(err)
	}
}

// Create Account
func (m *Manager) CreateAccount(w http.ResponseWriter, req *http.Request) {

	// Get request data
	vals, err := UrlValues(req)

	if err != nil {
		http.Error(w, "Failed to read request data", http.StatusBadRequest)
		return
	}

	username := vals.Get("username")
	password := vals.Get("password")

	// Create action
	action := NewAction(ActionCreateAccount, []byte(username))

	// Generate new keypair
	pubKey, privKey, err := GenerateKeypair(password)

	if err != nil {
		panic(err)
	}

	// Prepare and sign action
	action.Prepare(pubKey, 1) // pass sequence=1
	action.Sign(privKey, m.chainID)

	// Broadcast tx
	result, err := m.proxy.BroadcastTx("sync", action.Tx())

	if err != nil {
		ManagerRespond(w, MessageCreateAccount(err, nil, nil))
		return
	}

	if result.Code != 0 {
		ManagerRespond(w, MessageCreateAccount(nil, nil, result))
		return
	}

	keypair := NewKeypair(pubKey, privKey)

	ManagerRespond(w, MessageCreateAccount(nil, keypair, result))
}

func (m *Manager) RemoveAccount(w http.ResponseWriter, req *http.Request) {

	// Make sure we're logged in
	if m.acc == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	// Create action
	action := NewAction(ActionRemoveAccount, nil)

	// Prepare and sign action
	action.Prepare(m.acc.PubKey, m.acc.Sequence)
	action.Sign(m.acc.PrivKey, m.chainID)

	// Broadcast tx
	result, err := m.proxy.BroadcastTx("sync", action.Tx())

	if err != nil {
		ManagerRespond(w, MessageRemoveAccount(err, nil))
		return
	}

	ManagerRespond(w, MessageRemoveAccount(nil, result))
}

func (m *Manager) SubmitForm(w http.ResponseWriter, req *http.Request) {

	// Make sure we're logged in
	if m.acc == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	// Get request data // mimetype = "multipart/form-data"
	f, err := MultipartForm(req)

	if err != nil {
		http.Error(w, "Failed to read request data", http.StatusBadRequest)
		return
	}

	// Form // TODO: field validation
	var form Form

	// Info // check lengths of slices?
	form.Issue = f.Value["issue"][0]
	form.Location = f.Value["location"][0]
	form.Description = f.Value["description"][0]
	form.SubmittedAt = time.Now().Local().String()
	form.Submitter = PubKeytoHexstr(m.acc.PubKey)

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
	action := NewAction(ActionSubmitForm, wire.BinaryBytes(form))

	// Prepare and sign action
	m.logger.Info("Manager", "account_seq", m.acc.Sequence)
	action.Prepare(m.acc.PubKey, m.acc.Sequence+1)
	action.Sign(m.acc.PrivKey, m.chainID)

	// Broadcast tx
	result, err := m.proxy.BroadcastTx("sync", action.Tx())

	if err != nil {
		ManagerRespond(w, MessageSubmitForm(err, nil, nil))
		return
	}

	if result.Code != 0 {
		ManagerRespond(w, MessageSubmitForm(nil, nil, result))
		return
	}

	// CheckTx is ok so we can increment account sequence
	m.acc.Sequence++

	ManagerRespond(w, MessageSubmitForm(nil, form.ID(), result))
}

func (m *Manager) FindForm(w http.ResponseWriter, req *http.Request) {

	// Get values from request body
	vals, err := UrlValues(req)

	if err != nil {
		http.Error(w, "Failed to read request data", http.StatusBadRequest)
		return
	}

	formID, err := hex.DecodeString(vals.Get("form_id"))

	if err != nil || len(formID) != FORM_ID_LENGTH {
		http.Error(w, "Invalid form ID", http.StatusBadRequest)
		return
	}

	query := KeyQuery(formID)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageFindForm(err, nil, nil))
		return
	}

	tmResult := result.Result

	if tmResult.Code != 0 {
		ManagerRespond(w, MessageFindForm(nil, nil, result))
		return
	}

	// Form
	form := &Form{}

	err = wire.ReadBinaryBytes(tmResult.Data, form)

	if err != nil {
		panic(err)
	}

	ManagerRespond(w, MessageFindForm(nil, form, result))
}

func (m *Manager) BlockStream(done <-chan struct{}) {

	// Get latest block height
	status, err := m.proxy.GetStatus()
	if err != nil {
		// just set latest height to 0
		m.latestHeight = 0
	} else {
		m.latestHeight = status.LatestBlockHeight
	}

	m.logger.Info("Streaming blocks...", "start_height", m.latestHeight)

	for {

		select {

		case <-done:
			return

		default:

			status, err := m.proxy.GetStatus()

			if err != nil {
				//handle
				continue
			}

			if m.latestHeight == status.LatestBlockHeight {
				time.Sleep(time.Second * 5)
				continue
			} else if m.latestHeight > status.LatestBlockHeight {
				//shouldn't happen
			}

			// Get blocks
			for height := m.latestHeight; height <= status.LatestBlockHeight; height++ {
				block, err := m.proxy.GetBlock(height)
				if err != nil {
					//handle
					continue
				}
				m.blocks <- block.Block
			}

			m.latestHeight = status.LatestBlockHeight
		}
	}
}

func (m *Manager) Updates(w http.ResponseWriter, req *http.Request) {

	// Make sure we've logged in
	if m.acc == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	/*
		// Get values from request body
		vals, err := UrlValues(req)

		if err != nil {
			http.Error(w, "Failed to read request data", http.StatusBadRequest)
			return
		}

		issue := vals.Get("issue")
		pubKeystr := PubKeytoHexstr(m.acc.PubKey)

		var action Action
		var form Form
	*/

	// Subscribe to new block event
	err := m.proxy.SubscribeNewBlock()
	if err != nil {
		panic(err)
	}

	/*
		// Start block stream
		done := make(chan struct{})
		go m.BlockStream(done)

		block, ok := <-m.blocks

		if !ok {
			// Block channel closed
			// Shouldn't happen
			panic("Block channel closed")
		}
	*/

	for {

		m.logger.Info("Waiting for new block...")

		msg, err := m.proxy.ReadMessage()

		if err != nil {
			panic(err)
			// m.logger.Error(err.Error())
			// continue
		}

		fmt.Printf("%v\n", msg)

		/*
			m.logger.Info("New block!", "height", block.Height)

			for _, tx := range block.Txs {
				err := wire.ReadBinaryBytes(tx, &action)
				if err != nil {
					// Cannot decode tx bytes to action
					// Shouldn't happen
				}
				if action.Type != ActionSubmitForm {
					continue
				}
				err = wire.ReadBinaryBytes(action.Data, &form)
				if err != nil {
					// Cannot decode action data to form
					// Shouldn't happen
				}
				// Check if form is ours
				if form.Submitter == pubKeystr {
					// Send receipt
					m.logger.Info("Sending receipt...")
					receipt := NewReceipt(tx, block)
					update, err := NewUpdate(receipt)
					if err != nil {
						panic(err)
					}
					// Write update to websocket
					m.proxy.WriteWS("json", update)
				}
				if form.Issue != issue {
					continue
				}
				// Send form to feed
				update, err := NewUpdate(&form)
				if err != nil {
					panic(err)
				}
				m.logger.Info("Sending feed update...")
				m.proxy.WriteWS("json", update)
			}
		*/
	}
}

/*
func (am *Manager) Updates(w http.ResponseWriter, req *http.Request) {

	// Websocket
	ws, err := upgrader.Upgrade(w, req, nil)

	if err != nil {
		panic(err)
	}

	defer ws.Close()

	v := &struct {
		Issue     string `json:"issue"`
		PubKeystr string `json:"public_key"`
	}{}

	// Read from websocket
	err = ws.ReadJSON(v)

	if err != nil {
		panic(err) //for now
	}

	// Get conn to TMSP server

	am.ConnsMaction.Lock()
	conn, ok := am.Conns[v.PubKeystr]
	am.ConnsMaction.Unlock()

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
func (am *Manager) SearchForms(w http.ResponseWriter, req *http.Request) {

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

func (am *Manager) CheckMessages(w http.ResponseWriter, req *http.Request) {

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

func (am *Manager) SendMessage(w http.ResponseWriter, req *http.Request) {

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
func (am *Manager) CreateAdmin(w http.ResponseWriter, req *http.Request) {

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	// Create action
	var action types.Action
	action.Type = types.CreateAdminTx

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
		action.Data = buf.Bytes()

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

		action.SetSequence(acc.Sequence)
		action.SetAccount(pubKey)

		reqQuery = tmsp.ToRequestQuery([]byte{QueryChainID})
		res, err = am.WriteRequest(reqQuery)

		if err != nil {
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		chainID := res.GetQuery().Log
		action.SetSignature(privKey, chainID)

		// TxBytes in AppendTx request
		buf = new(bytes.Buffer)
		wire.WriteBinary(tx, buf, &n, &err)
		reqAppendTx := tmsp.ToRequestAppendTx(buf.Bytes())
		res, err = am.WriteRequest(reqAppendTx)

		resAppendTx := res.GetAppendTx()

		if resAppendTx.Code != 0  {
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

func (am *Manager) RemoveAdmin(w http.ResponseWriter, req *http.Request) {

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	// Create tx
	var action types.Action
	action.Type = types.RemoveAdminTx

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

		action.SetSequence(acc.Sequence)
		action.SetAccount(pubKey)

		reqQuery = tmsp.ToRequestQuery([]byte{QueryChainID})
		res, err = am.WriteRequest(reqQuery)

		if err != nil {
			ws.WriteMessage(ws.TextMessage, []byte(read_response_failure))
			continue
		}

		chainID := res.GetQuery().Log
		action.SetSignature(privKey, chainID)

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

		if resAppendTx.Code != 0  {
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
