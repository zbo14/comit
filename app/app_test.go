package app

import (
	"bufio"
	"bytes"
	"flag"
	. "github.com/tendermint/go-common"
	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/server"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/lib"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"reflect"
	"testing"
)

const (
	CHAIN_ID = "testing"

	ISSUE_1       = "citizen complaint"
	LOCATION_1    = "somewhere"
	DESCRIPTION_1 = "something happened"

	ISSUE_2       = "incident report"
	LOCATION_2    = "somewhere else"
	DESCRIPTION_2 = "something else happened"

	PUBKEY_LENGTH  = 32
	PRIVKEY_LENGTH = 64
	MOMENT_LENGTH  = 32
	FORM_ID_LENGTH = 16
)

func TestStream(t *testing.T) {

	cliPtr := flag.String("cli", "local", "Client address, or 'local' for embedded")
	flag.Parse()

	// Client
	cli, err := NewClient(*cliPtr, "socket")
	if err != nil {
		Exit("connect to client: " + err.Error())
	}

	// Create comit app
	app := NewApp(cli)
	app.SetOption("base/chainID", CHAIN_ID)
	app.state.PrintStore()

	// Constituent
	privKey := crypto.GenPrivKeyEd25519()
	pubKey := privKey.PubKey().(crypto.PubKeyEd25519)
	account := types.NewAccount(pubKey, 0)
	app.state.SetAccount(account.PubKey.Address(), account)

	privKeyString := BytesToHexString(privKey[:])
	pubKeyString := BytesToHexString(pubKey[:])

	// Admin
	adminPrivKey := crypto.GenPrivKeyEd25519()
	adminPubKey := adminPrivKey.PubKey().(crypto.PubKeyEd25519)
	admin := types.NewAdmin(adminPubKey)
	app.state.SetAccount(admin.PubKey.Address(), admin)

	adminPrivKeyString := BytesToHexString(adminPrivKey[:])
	adminPubKeyString := BytesToHexString(adminPubKey[:])

	// Start listener
	server, err := server.NewSocketServer("unix://test.sock", app)
	if err != nil {
		Exit(err.Error())
	}
	defer server.Stop()
	conn, err := Connect("unix://test.sock")
	if err != nil {
		Exit(err.Error())
	}

	numTxs := 4

	//-------- Read Responses --------//

	go func() {
		r := bufio.NewReader(conn)
		for i := 0; i < numTxs; i++ {
			var res = &tmsp.Response{}
			err := tmsp.ReadMessage(r, res)
			if err != nil {
				Exit(err.Error())
			}
			switch r := res.Value.(type) {
			case *tmsp.Response_AppendTx:
				if r.AppendTx.Code != tmsp.CodeType_OK {
					t.Error("AppendTx failed with ret_code", r.AppendTx.Code)
				}
			case *tmsp.Response_Query:
				if r.Query.Code != tmsp.CodeType_OK {
					t.Error("Query failed with ret_code", r.Query.Code)
				}
			case *tmsp.Response_Flush:
				// ignore
			default:
				t.Error("Unexpected response type", reflect.TypeOf(res.Value))
			}
		}
	}()

	//-------- Execute Requests, Write Responses --------//

	w := bufio.NewWriter(conn)

	// SubmitTxs
	formID_1 := SubmitTx(ISSUE_1, LOCATION_1, DESCRIPTION_1,
		pubKeyString, privKeyString, app, w, t)

	formID_2 := SubmitTx(ISSUE_2, LOCATION_2, DESCRIPTION_2,
		pubKeyString, privKeyString, app, w, t)

	// ResolveTx
	ResolveTx(formID_1, adminPubKeyString, adminPrivKeyString, app, w, t)
	ResolveTx(formID_2, adminPubKeyString, adminPrivKeyString, app, w, t)

	// Send final flush message
	err = tmsp.WriteMessage(tmsp.ToRequestFlush(), conn)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func CreateAccountTx(secret string, app *App, w *bufio.Writer, t *testing.T) (string, string) {

	// Create tx
	var tx = types.Tx{}
	tx.Type = types.CreateAccountTx

	// Secret
	tx.Data = []byte(secret)

	// Set Sequence, Account, Signature
	pubKey, privKey := CreateKeys(tx.Data)
	tx.SetSequence(0)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, CHAIN_ID)

	// TxBytes in AppendTx request
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, buf, &n, &err)
	result := app.AppendTx(buf.Bytes())
	res := tmsp.ToResponseAppendTx(result.Code, result.Data, result.Log)
	err = tmsp.WriteMessage(res, w)
	if err != nil {
		t.Fatal(err)
	}
	pubKeyString := BytesToHexString(pubKey[:])
	privKeyString := BytesToHexString(privKey[:])
	return pubKeyString, privKeyString
}

func SubmitTx(issue, location, description, pubKeyString, privKeyString string,
	app *App, w *bufio.Writer, t *testing.T) string {

	// Create tx
	var tx = types.Tx{}
	tx.Type = types.SubmitTx

	// Keys
	var pubKey = crypto.PubKeyEd25519{}
	pubKeyBytes, _ := HexStringToBytes(pubKeyString)
	copy(pubKey[:], pubKeyBytes)

	var privKey = crypto.PrivKeyEd25519{}
	privKeyBytes, _ := HexStringToBytes(privKeyString)
	copy(privKey[:], privKeyBytes)

	// Form
	form, _ := lib.MakeForm(issue, location, description, pubKeyString)
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(*form, buf, &n, &err)
	tx.Data = buf.Bytes()

	// Set Sequence, Account, Signature
	addr := pubKey.Address()
	seq, err := app.GetSequence(addr)
	if err != nil {
		t.Fatal(err)
	}
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, app.GetChainID())

	// TxBytes in AppendTx
	buf = new(bytes.Buffer)
	wire.WriteBinary(tx, buf, &n, &err)
	result := app.AppendTx(buf.Bytes())
	res := tmsp.ToResponseAppendTx(result.Code, result.Data, result.Log)
	err = tmsp.WriteMessage(res, w)
	if err != nil {
		t.Fatal(err)
	}
	formID := BytesToHexString(form.ID())
	return formID
}

func ResolveTx(formID, pubKeyString, privKeyString string,
	app *App, w *bufio.Writer, t *testing.T) {

	// Create Tx
	var tx = types.Tx{}
	tx.Type = types.ResolveTx

	// FormID
	IDbytes, err := HexStringToBytes(formID)
	if err != nil {
		t.Fatal(err.Error())
	}
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(IDbytes, buf, &n, &err)
	tx.Data = buf.Bytes()

	// PubKey
	var pubKey = crypto.PubKeyEd25519{}
	pubKeyBytes, _ := HexStringToBytes(pubKeyString)
	copy(pubKey[:], pubKeyBytes)

	// PrivKey
	var privKey = crypto.PrivKeyEd25519{}
	privKeyBytes, _ := HexStringToBytes(privKeyString)
	copy(privKey[:], privKeyBytes)

	addr := pubKey.Address()
	seq, err := app.GetSequence(addr)
	if err != nil {
		t.Fatal(err)
	}
	// Set Sequence, Account, Signature
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, CHAIN_ID)

	// TxBytes in AppendTx request
	buf = new(bytes.Buffer)
	wire.WriteBinary(tx, buf, &n, &err)
	result := app.AppendTx(buf.Bytes())
	res := tmsp.ToResponseAppendTx(result.Code, result.Data, result.Log)
	err = tmsp.WriteMessage(res, w)
	if err != nil {
		t.Fatal(err)
	}
	return
}
