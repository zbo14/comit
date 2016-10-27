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
	ntwk "github.com/zballs/comit/network"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"log"
	"net/http"
)

type ActionManager struct {
	app      *app.App
	network  *p2p.Switch
	peerAddr string
}

func CreateActionManager(app_ *app.App, network *p2p.Switch, peerAddr string) *ActionManager {
	return &ActionManager{
		app:      app_,
		network:  network,
		peerAddr: peerAddr,
	}
}

// Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Issues
func (am *ActionManager) GetIssues(w http.ResponseWriter, req *http.Request) {

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Panic(err)
	}

	issues := am.app.Issues()
	var buf bytes.Buffer
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
	tx.Data = secret

	// Set Sequence, Account, Signature
	pubKey, privKey := CreateKeys(tx.Data) // create keys now
	tx.SetSequence(0)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, am.app.GetChainID())

	// TxBytes in AppendTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	res := am.app.AppendTx(txBuf.Bytes())

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
		msg := Fmt(invalid_hex, pubKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
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
		msg := Fmt(invalid_hex, privKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
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

func (am *ActionManager) ConnectAccount(w http.ResponseWriter, req *http.Request) {

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
		msg := Fmt(invalid_hex, pubKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		conn.Close()
	}
	pubKeyBytes = pubKeyBytes[:n]
	var pubKey crypto.PubKeyEd25519
	copy(pubKey[:], pubKeyBytes[:])

	// Check if already connected
	log.Println(am.network.Peers().List())
	pubKeyString := BytesToHexString(pubKeyBytes)
	peer := am.network.Peers().Get(pubKeyString)
	if peer != nil {
		log.Println("Peer already connected")
		conn.WriteMessage(websocket.TextMessage, []byte(already_connected))
		conn.Close()
		return
	}

	// PrivKey
	_, rawBytes, err = conn.ReadMessage()
	if err != nil {
		log.Panic(err)
	}
	privKeyBytes := make([]byte, 64)
	n, err = hex.Decode(privKeyBytes, rawBytes)
	if err != nil {
		msg := Fmt(invalid_hex, privKeyBytes)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
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

		// Peer switch info
		peer_sw := p2p.NewSwitch(ntwk.Config)
		peer_sw.SetNodeInfo(&p2p.NodeInfo{
			Network: "testing",
			Version: "311.311.311",
		})
		peer_sw.SetNodePrivKey(privKey)

		// Add reactors
		feeds := am.network.Reactor("feeds").(*ntwk.MyReactor)
		peer_sw.AddReactor("feeds", feeds)
		messages := am.network.Reactor("messages").(*ntwk.MyReactor)
		peer_sw.AddReactor("messages", messages)

		// Add listener
		l := p2p.NewDefaultListener("tcp", am.peerAddr, false)
		peer_sw.AddListener(l)
		peer_sw.Start()

		// Add peer to network
		addr := p2p.NewNetAddressString(am.peerAddr)
		_, err := am.network.DialPeerWithAddress(addr)

		if err != nil {
			log.Println(err.Error())
			conn.WriteMessage(websocket.TextMessage, []byte(connect_failure))
			conn.Close()
		} else {
			log.Println("SUCCESS peer connected to network")
			conn.WriteMessage(websocket.TextMessage, []byte("connected"))
			conn.Close()
		}
	}
}
