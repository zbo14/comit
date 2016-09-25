package actions

import (
	"bytes"
	. "github.com/tendermint/go-common"
	// . "github.com/tendermint/go-crypto"
	"github.com/tendermint/tmsp/server"
	"github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	. "github.com/zballs/3ii/types"
	// . "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestStream(t *testing.T) {

	numAppendTxs := 8
	app := NewApplication()
	server, err := server.NewSocketServer("unix://test.sock", app)
	if err != nil {
		Exit(err.Error())
	}
	defer server.Stop()
	conn, err := Connect("unix://test.sock")
	if err != nil {
		Exit(err.Error())
	}

	// Start the action listener
	_, err = StartActionListener()
	if err != nil {
		Exit(err.Error())
	}

	// recvr := CreateSwitch(GenPrivKeyEd25519())

	userPubKeyString, _, err := app.UserManager().Register("user-passphrase")
	if err != nil {
		Exit(err.Error())
	}

	adminPubKeyString, _, err := app.AdminManager().Register(
		"infrastructure",
		"head of transportation",
		"admin-passphrase",
	)
	if err != nil {
		Exit(err.Error())
	}

	// Read response data
	done := make(chan struct{}, 1)
	go func() {
		iter := 0
		for {
			var res = &types.Response{}
			err := types.ReadMessage(conn, res)
			if err != nil {
				Exit(err.Error())
			}

			// Process response
			switch r := res.Value.(type) {
			case *types.Response_AppendTx:
				iter += 1
				if r.AppendTx.Code != types.CodeType_OK {
					t.Error("AppendTx failed with ret_code", r.AppendTx.Code)
				}
				if iter > numAppendTxs {
					t.Fatal("Too many responses")
				}
				t.Log("appendTx", iter)
				if iter == numAppendTxs {
					go func() {
						time.Sleep(time.Second * 2) // Wait for a bit to allow iter overflow
						close(done)
					}()
				}
			case *types.Response_Query:
				if r.Query.Code != types.CodeType_OK {
					t.Error("Query failed with ret_code", r.Query.Code)
				}
				t.Log("query")
			case *types.Response_Flush:
				// ignore
			default:
				t.Error("Unexpected response type", reflect.TypeOf(res.Value))
			}
		}
	}()

	// Write requests
	for iter := 0; iter < numAppendTxs; iter++ {

		// Generate submit string
		submit, ID := GenerateSubmit(userPubKeyString)

		// Send request
		var req = types.ToRequestAppendTx([]byte(submit))
		err := types.WriteMessage(req, conn)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Generate resolve string
		resolve := GenerateResolve(ID, adminPubKeyString)

		// Send request
		req = types.ToRequestAppendTx([]byte(resolve))
		err = types.WriteMessage(req, conn)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Sometimes send flush messages
		if iter%17 == 0 {
			t.Log("flush")
			err := types.WriteMessage(types.ToRequestFlush(), conn)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	}

	// Send final flush message
	err = types.WriteMessage(types.ToRequestFlush(), conn)
	if err != nil {
		t.Fatal(err.Error())
	}

	<-done
}

const (
	numbers = "0123456789"
	letters = "abcdefghijklmnopqrstuvwxyz"
	symbols = ".,?!-/"
)

var services = lib.SERVICE.Services()

func RandomChar(str string) string {
	return string(str[rand.Intn(len(str))])
}

func RandomChars(str string, n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		buf.WriteString(RandomChar(str))
	}
	return buf.String()
}

func RandomString(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

func RandOption(serviceName string) string {
	sd := lib.SERVICE.ServiceDetail(serviceName)
	if sd == nil {
		return ""
	}
	return RandomString(sd.Options())
}

func GenerateAddress() string {
	return RandomChars(numbers, 2) + " " + RandomChars(letters, 6) + " st."
}

func GenerateSubmit(pubKeyString string) (string, string) {
	action := lib.SERVICE.WriteField("submit", "action")
	serviceName := RandomString(services)
	service := lib.SERVICE.WriteField(serviceName, "service")
	address := lib.SERVICE.WriteField(GenerateAddress(), "address")
	description := lib.SERVICE.WriteField("this is a description", "description")
	detail := lib.SERVICE.WriteDetail(RandOption(serviceName), serviceName)
	pubkey := util.WritePubKeyString(pubKeyString)
	str := action + service + address + description + detail + pubkey
	form, _ := MakeForm(str)
	return str, FormID(form)
}

func GenerateResolve(ID string, pubKeyString string) string {
	action := lib.SERVICE.WriteField("resolve", "action")
	formID := lib.SERVICE.WriteField(ID, "ID")
	pubkey := util.WritePubKeyString(pubKeyString)
	return action + formID + pubkey
}

// example
// "service {street light out}address {25 Howard St.}description {the light is out.}completely out? {yes}pubkey {19e28ed0accfa56c9f457c943fb8c0fd324d4600429cb22fbfd5cd303d95687e}"
