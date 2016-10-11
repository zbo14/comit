package app

import (
	"bytes"
	"flag"
	. "github.com/tendermint/go-common"
	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/server"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/3ii/lib"
	"github.com/zballs/3ii/types"
	. "github.com/zballs/3ii/util"
	"net"
	"reflect"
	"testing"
)

const (
	chainID     = "testing"
	seq         = 1
	service     = "street light out"
	address     = "25 Howard Street"
	description = "the light is out"
	detail      = "yes"
)

func TestStream(t *testing.T) {

	cliPtr := flag.String("cli", "local", "Client address, or 'local' for embedded")
	flag.Parse()

	// Client
	cli, err := NewClient(*cliPtr, "socket")
	if err != nil {
		Exit("connect to client: " + err.Error())
	}

	// Create 3ii app
	app_ := NewApp(cli)
	app_.SetOption("base/chainID", chainID)

	// Start listener
	server, err := server.NewSocketServer("unix://test.sock", app_)
	if err != nil {
		Exit(err.Error())
	}
	defer server.Stop()
	conn, err := Connect("unix://test.sock")
	if err != nil {
		Exit(err.Error())
	}

	//-------- Read response data --------//

	go func() {
		for {
			var res = &tmsp.Response{}
			err := tmsp.ReadMessage(conn, res)
			if err != nil {
				Exit(err.Error())
			}

			// Process response
			switch r := res.Value.(type) {
			case *tmsp.Response_AppendTx:
				if r.AppendTx.Code != tmsp.CodeType_OK {
					t.Error("AppendTx failed with ret_code", r.AppendTx.Code)
				}
				t.Log("appendTx")
			case *tmsp.Response_Query:
				if r.Query.Code != tmsp.CodeType_OK {
					t.Error("Query failed with ret_code", r.Query.Code)
				}
				t.Log("query")
			case *tmsp.Response_Flush:
				// ignore
			default:
				t.Error("Unexpected response type", reflect.TypeOf(res.Value))
			}
		}
	}()

	//-------- Write Requests --------//

	// CreateAccountTx
	pubKeyString, privKeyString := CreateAccountTx("marmalade", conn, t)

	// SubmitTx
	req := SubmitTx(service, address, description, detail, pubKeyString, privKeyString, conn, t)
	t.Log(req.String())

	// Create form
	form := &lib.Form{
		Posted:      TimeString(),
		Service:     service,
		Address:     address,
		Description: description,
		Detail:      detail,
		Status:      "unresolved",
	}

	// Get formID
	formID := BytesToHexString(form.ID())

	// ResolveTx
	req = ResolveTx(formID, pubKeyString, privKeyString, conn, t)
	t.Log(req.String())

	// Send final flush message
	err = tmsp.WriteMessage(tmsp.ToRequestFlush(), conn)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func CreateAccountTx(secret string,
	conn net.Conn, t *testing.T) (string, string) {

	// Create tx
	var tx = types.Tx{}
	tx.Type = types.CreateAccountTx

	// Secret
	tx.Data = []byte(secret)

	// Set Sequence, Account, Signature
	pubKey, privKey := CreateKeys(tx.Data)
	tx.Input.Sequence = 1
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, chainID)

	// TxBytes in AppendTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	// res := app_.AppendTx(txBuf.Bytes())
	req := tmsp.ToRequestAppendTx(txBuf.Bytes())
	err = tmsp.WriteMessage(req, conn)
	if err != nil {
		t.Fatal(err.Error())
	}
	pubKeyString := BytesToHexString(pubKey[:])
	privKeyString := BytesToHexString(privKey[:])
	return pubKeyString, privKeyString
}

func SubmitTx(service, address, description, detail, pubKeyString, privKeyString string,
	conn net.Conn, t *testing.T) *tmsp.Request {

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
		t.Fatal(err.Error())
	}
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	var privKey = crypto.PrivKeyEd25519{}
	privKeyBytes, err := HexStringToBytes(privKeyString)
	if err != nil {
		t.Fatal(err.Error())
	}
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, chainID)

	// Increment sequence
	seq += 1

	// TxBytes in AppendTx
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	req := tmsp.ToRequestAppendTx(txBuf.Bytes())
	err = tmsp.WriteMessage(req, conn)
	if err != nil {
		t.Fatal(err.Error())
	}
	return req
}

func ResolveTx(formID, pubKeyString, privKeyString string,
	conn net.Conn, t *testing.T) *tmsp.Request {

	// Create Tx
	var tx = types.Tx{}
	tx.Type = types.ResolveTx

	// FormID
	formID_bytes, err := HexStringToBytes(formID)
	if err != nil {
		t.Fatal(err.Error())
	}
	tx.Data = formID_bytes

	// PubKey
	var pubKey = crypto.PubKeyEd25519{}
	pubKeyBytes, err := HexStringToBytes(pubKeyString)
	if err != nil {
		t.Fatal(err.Error())
	}
	copy(pubKey[:], pubKeyBytes[:])

	// PrivKey
	var privKey = crypto.PrivKeyEd25519{}
	privKeyBytes, err := HexStringToBytes(privKeyString)
	if err != nil {
		t.Fatal(err.Error())
	}
	copy(privKey[:], privKeyBytes[:])

	// Set Sequence, Account, Signature
	tx.SetSequence(seq)
	tx.SetAccount(pubKey)
	tx.SetSignature(privKey, chainID)

	// Increment sequence
	seq += 1

	// TxBytes in AppendTx request
	txBuf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(tx, txBuf, &n, &err)
	req := tmsp.ToRequestAppendTx(txBuf.Bytes())
	err = tmsp.WriteMessage(req, conn)
	if err != nil {
		t.Fatal(err.Error())
	}
	return req
}
