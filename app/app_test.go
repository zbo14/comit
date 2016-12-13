package app

import (
	"bufio"
	"flag"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/server"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/state"
	. "github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"reflect"
	"testing"
	"time"
)

const (
	chainID = "testing"

	username = "someone"
	password = "canyouguess?"

	issue1       = "citizen complaint"
	location1    = "somewhere"
	description1 = "something happened"

	issue2       = "incident report"
	location2    = "somewhere else"
	description2 = "something else happened"
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
	app.SetOption("base/chainID", chainID)

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

	numReqs := 7

	// Read responses

	go func() {
		r := bufio.NewReader(conn)
		for i := 0; i < numReqs; i++ {
			var res = &tmsp.Response{}
			err := tmsp.ReadMessage(r, res)
			if err != nil {
				t.Error("Error reading response", err)
				continue
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

	// Execute requests // Write responses

	w := bufio.NewWriter(conn)

	// (1) Create account
	pubKey, privKey := CreateAccount(username, password, app, w, t)

	// (2) Query account
	acc := QueryAccount(pubKey, app, w, t)

	// Create private account
	privAcc := NewPrivAccount(acc, privKey)

	// Increment sequence for next appendTx request
	privAcc.Sequence++

	// (3,4) Submit forms
	formID1 := SubmitForm(issue1, location1, description1, privAcc, app, w, t)
	fmt.Printf("%X\n", formID1)
	privAcc.Sequence++
	formID2 := SubmitForm(issue2, location2, description2, privAcc, app, w, t)
	fmt.Printf("%X\n", formID2)

	// (5,6) Query forms
	form1 := QueryForm(formID1, app, w, t)
	form2 := QueryForm(formID2, app, w, t)
	fmt.Println(form1)
	fmt.Println(form2)

	// (7) Send flush message
	err = tmsp.WriteMessage(tmsp.ToRequestFlush(), conn)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func CreateAccount(username, password string, app *App, w *bufio.Writer, t *testing.T) (crypto.PubKey, crypto.PrivKey) {

	// Create keys
	pubKey, privKey, err := GenerateKeypair(password)
	if err != nil {
		t.Fatal(err)
	}

	// Create action
	data := []byte(username)
	action := NewAction(ActionCreateAccount, data)

	// Prepare and sign action
	action.Prepare(pubKey, 1)
	action.Sign(privKey, chainID)

	result := app.AppendTx(action.Tx())
	response := tmsp.ToResponseAppendTx(result.Code, result.Data, result.Log)
	err = tmsp.WriteMessage(response, w)
	if err != nil {
		t.Fatal(err)
	}
	return pubKey, privKey
}

func QueryAccount(pubKey crypto.PubKey, app *App, w *bufio.Writer, t *testing.T) *Account {

	// Create query
	accKey := state.AccountKey(pubKey.Address())
	query := KeyQuery(accKey)

	// Query account
	result := app.Query(query)
	response := tmsp.ToResponseQuery(result.Code, result.Data, result.Log)
	err := tmsp.WriteMessage(response, w)
	if err != nil {
		t.Fatal(err)
	}

	var acc *Account
	err = wire.ReadBinaryBytes(result.Data, &acc)
	if err != nil {
		t.Fatal(err)
	}

	return acc
}

func SubmitForm(issue, location, description string, privAcc *PrivAccount, app *App, w *bufio.Writer, t *testing.T) []byte {

	// Create form
	form := Form{
		Description: description,
		Issue:       issue,
		Location:    location,
		SubmittedAt: time.Now().Local().String(),
		Submitter:   PubKeytoHexstr(privAcc.PubKey),
	}

	// Create action
	data := wire.BinaryBytes(form)
	action := NewAction(ActionSubmitForm, data)

	// Prepare and sign action
	action.Prepare(privAcc.PubKey, privAcc.Sequence)
	action.Sign(privAcc.PrivKey, chainID)

	result := app.AppendTx(action.Tx())
	response := tmsp.ToResponseAppendTx(result.Code, result.Data, result.Log)
	err := tmsp.WriteMessage(response, w)
	if err != nil {
		t.Fatal(err)
	}
	return form.ID()
}

func QueryForm(formID []byte, app *App, w *bufio.Writer, t *testing.T) Form {

	// Create query
	query := KeyQuery(formID)

	// Query form
	result := app.Query(query)
	response := tmsp.ToResponseQuery(result.Code, result.Data, result.Log)
	err := tmsp.WriteMessage(response, w)
	if err != nil {
		t.Fatal(err)
	}

	var form Form
	err = wire.ReadBinaryBytes(result.Data, &form)
	if err != nil {
		t.Fatal(err)
	}
	return form
}
