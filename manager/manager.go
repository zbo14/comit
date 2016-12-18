package manager

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-ipfs/blocks"
	core "github.com/ipfs/go-ipfs/core"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/pkg/errors"
	"github.com/tendermint/go-merkle"
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
	proxy *Proxy
	// Proxy mtx necessary?

	logger Logger

	acc     *PrivAccount
	chainID string
	issues  []string

	latestHeight int
	blocks       chan *tndr.Block

	node   *core.IpfsNode
	cancel context.CancelFunc
}

func CreateManager(remote string) *Manager {
	return &Manager{
		proxy:   NewProxy(remote, "/websocket"),
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

// IPFS Node
func (m *Manager) InitNode() error {
	if m.node != nil {
		return errors.New("Node already exists")
	}
	r, err := fsrepo.Open("~/.ipfs")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &core.BuildCfg{
		Repo:   r,
		Online: true,
	}

	m.node, err = core.NewNode(ctx, cfg)

	if err != nil {
		return err
	}

	m.cancel = cancel
	return nil
}

// ChainID

func (m *Manager) GetChainID(w http.ResponseWriter, req *http.Request) {

	query := EmptyQuery(QueryChainID)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageChainID(err))
		return
	}

	err = ResultToError(result)

	if err == nil {
		data, _, err := wire.GetByteSlice(result.Result.Data)
		if err == nil {
			m.chainID = string(data)
		}
	}

	ManagerRespond(w, MessageChainID(err))
}

// Issues

func (m *Manager) GetIssues(w http.ResponseWriter, req *http.Request) {

	query := EmptyQuery(QueryIssues)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageIssues(nil, err))
		return
	}

	err = ResultToError(result)

	if err != nil {
		ManagerRespond(w, MessageIssues(nil, err))
		return
	}

	var issues []string
	wire.ReadBinaryBytes(result.Result.Data, &issues)
	m.issues = issues

	ManagerRespond(w, MessageIssues(m.issues, nil))
}

func (m *Manager) Issues(w http.ResponseWriter, req *http.Request) {

	if len(m.issues) == 0 {
		m.GetIssues(w, req)
		return
	}

	ManagerRespond(w, MessageIssues(m.issues, nil))
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
	query := KeyQuery(acckey, QueryValue)

	result, err := m.proxy.TMSPQuery(query)

	if err == nil {
		err = ResultToError(result)
		if err == nil {
			var acc *Account
			wire.ReadBinaryBytes(result.Result.Data, &acc)
			m.acc = NewPrivAccount(acc, privKey)
		}
	}

	ManagerRespond(w, MessageLogin(err))

	// Init IPFS node // check for err
	m.InitNode()

	// Start proxy ws
	m.proxy.StartWS()
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
		ManagerRespond(w, MessageCreateAccount(nil, err))
		return
	}

	err = ResultToError(result)

	if err != nil {
		ManagerRespond(w, MessageCreateAccount(nil, err))
		return
	}

	keypair := NewKeypair(pubKey, privKey)

	ManagerRespond(w, MessageCreateAccount(keypair, nil))
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

	if err == nil {
		err = ResultToError(result)
	}

	ManagerRespond(w, MessageRemoveAccount(err))
}

func (m *Manager) SubmitForm(w http.ResponseWriter, req *http.Request) {

	// Make sure we're logged in and node is running
	if m.acc == nil || m.node == nil {
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

	// Text // check lengths of slices?
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

	// Add IPFS block with form data
	b := blocks.NewBlock(wire.BinaryBytes(form))
	cid, err := m.node.Blocks.AddBlock(b)
	if err != nil {
		panic(err)
	}

	// Encode form info
	data := wire.BinaryBytes(NewInfo(cid, form))

	// Create action
	action := NewAction(ActionSubmitForm, data)

	// Prepare and sign action
	action.Prepare(m.acc.PubKey, m.acc.Sequence+1)
	action.Sign(m.acc.PrivKey, m.chainID)

	// Broadcast tx
	result, err := m.proxy.BroadcastTx("sync", action.Tx())

	if err != nil {
		ManagerRespond(w, MessageSubmitForm(nil, err))
		return
	}

	err = ResultToError(result)

	if err != nil {
		ManagerRespond(w, MessageSubmitForm(nil, err))
		return
	}

	// CheckTx is ok so we can increment sequence
	m.acc.Sequence++

	idpair := NewIdpair(form, cid)
	ManagerRespond(w, MessageSubmitForm(idpair, nil))
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

	query := KeyQuery(formID, QueryValue)

	result, err := m.proxy.TMSPQuery(query)

	if err != nil {
		ManagerRespond(w, MessageFindForm(nil, err))
		return
	}

	err = ResultToError(result)

	if err != nil {
		ManagerRespond(w, MessageFindForm(nil, err))
		return
	}

	// Form
	form := &Form{}
	wire.ReadBinaryBytes(result.Result.Data, form)

	ManagerRespond(w, MessageFindForm(form, nil))
}

func (m *Manager) BlockStream(done <-chan struct{}) {

	// Subscribe to new block event
	err := m.proxy.SubscribeNewBlock()
	if err != nil {
		panic("Failed to subscribe to new block event")
	}

	// Get latest block height
	status, err := m.proxy.GetStatus()
	if err == nil {
		m.latestHeight = status.LatestBlockHeight
	}

	var block *tndr.Block
	var evDataBlock tndr.EventDataNewBlock
	var evData tndr.TMEventData

	m.logger.Info("Streaming blocks...", "start_height", m.latestHeight)

	for {

		select {

		case <-done:
			err := m.proxy.UnsubscribeNewBlock()
			if err != nil {
				panic("Failed to unsubscribe from new block event")
			}
			m.logger.Info("Done streaming blocks")
			return

		default:
			evData, err = m.proxy.ReadResult("NewBlock", &evDataBlock)
			if err != nil {
				panic(err)
			}

			switch evData.(type) {

			case tndr.EventDataNewBlock:
				block = evData.(tndr.EventDataNewBlock).Block
			case *tndr.EventDataNewBlock:
				block = evData.(*tndr.EventDataNewBlock).Block
			}

			if block == nil {
				continue
			}

			if m.latestHeight == block.Height {
				time.Sleep(time.Second * 5)
				continue
			} else if m.latestHeight > block.Height {
				//shouldn't happen
			} else if m.latestHeight+1 < block.Height {
				// missed block(s)
				m.logger.Warn("Missed block(s)", "missed", block.Height-m.latestHeight)
				// query missed blocks
				// should this run in goroutine and
				// should next read wait on its completion??
				for h := m.latestHeight + 1; h <= block.Height; h++ {
					result, err := m.proxy.GetBlock(h)
					if err != nil {
						panic(err)
					}
					m.blocks <- result.Block
				}
			}

			m.blocks <- block

			m.latestHeight = block.Height
		}
	}
}

func (m *Manager) Updates(w http.ResponseWriter, req *http.Request) {

	// This routine writes feed updates and blockchain receipts to ws
	// What information should receipts include? Root hash, form ID, anything else?

	// Websocket connection
	ws, err := Upgrader().Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}

	// Make sure we've logged in
	if m.acc == nil {
		ws.WriteMessage(websocket.TextMessage, []byte("Not logged in"))
		return
	}

	// Get values from request body
	_, data, err := ws.ReadMessage()
	if err != nil {
		panic(err)
	}

	issue := string(data)
	pubKeystr := PubKeytoHexstr(m.acc.PubKey)

	var action Action
	var form Form
	var proof merkle.IAVLProof
	var receipt *Receipt

	// Start block stream
	done := make(chan struct{})
	go m.BlockStream(done)

	for {

		m.logger.Info("Waiting for block...")

		block, ok := <-m.blocks

		if !ok {
			// Block channel closed
			// shouldn't happen
			panic("Block channel closed")
		}

		if receipt != nil {
			fmt.Println(receipt.BlockHeight, block.Height)
			// We have a receipt to send
			if receipt.BlockHeight >= block.Height {
				//shouldn't happen
			} else if receipt.BlockHeight+1 == block.Height {
				// Good, increment height
				receipt.BlockHeight++
			} else {
				// We missed blocks
				// Ok, set height anyway
				receipt.BlockHeight = block.Height
			}
			// Query merkle proof for committed form and verify receipt
			key := HexstrToBytes(receipt.FormID)
			query := KeyQuery(key, QueryProof)
			result, err := m.proxy.TMSPQuery(query)
			if err == nil {
				err = ResultToError(result)
				if err == nil {
					err = wire.ReadBinaryBytes(result.Result.Data, &proof)
					if err == nil {
						value := wire.BinaryBytes(form)
						verified := proof.Verify(key, value, block.AppHash)
						if verified {
							// Set app hash and write to ws
							receipt.AppHash = block.AppHash
							update, _ := NewUpdate(receipt, nil)
							ws.WriteJSON(update)
						} else {
							err = errors.New("Failed to verify receipt")
						}
					}
				}
			}

			if err != nil {
				panic(err)
				update, _ := NewUpdate(receipt, err)
				ws.WriteJSON(update)
			}
			receipt = nil
		}

		if len(block.Txs) == 0 {
			// m.logger.Warn("Block has no txs", "height", block.Height)
			continue
		}

		for _, tx := range block.Txs {
			err = wire.ReadBinaryBytes(tx, &action)
			if err == nil {
				if action.Type != ActionSubmitForm {
					continue
				}
				err = wire.ReadBinaryBytes(action.Data, &form)
				if err == nil {
					if form.Submitter == pubKeystr {
						// Create new receipt, do not set app hash yet
						// Once we recv next block we will send receipt
						receipt = NewReceipt(block.Height, form.ID())
					}
					if form.Issue != issue {
						// Not what we're looking for..
						continue
					}
					// Send form to feed
					update, _ := NewUpdate(&form, err)
					ws.WriteJSON(update)
				}
			}
		}
	}
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
*/
