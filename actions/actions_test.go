package actions

import (
	"bytes"
	. "github.com/tendermint/go-common"
	. "github.com/tendermint/go-crypto"
	"github.com/tendermint/tmsp/server"
	"github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/app"
	lib "github.com/zballs/3ii/lib"
	. "github.com/zballs/3ii/network"
	util "github.com/zballs/3ii/util"
	// "log"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestStream(t *testing.T) {

	numAppendTxs := 2
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

	recvrPrivKey := GenPrivKeyEd25519()
	// sendrPrivKey := GenPrivKeyEd25519()

	recvr := CreateSwitch(recvrPrivKey, "recr")
	// sendr := CreateSwitch(sendrPrivKey, "sendr")

	userPrivKey := GenPrivKeyEd25519()
	// log.Println("user privkey " + util.PrivKeyToString(userPrivKey))

	user := CreateSwitch(userPrivKey, "abc")
	_, _, err = app.AdminManager().RegisterUser(user.NodeInfo().Other[0], recvr)
	if err != nil {
		Exit(err.Error())
	}

	/*
		adminPrivKey := GenPrivKeyEd25519()
		log.Println("admin privkey " + util.PrivKeyToString(adminPrivKey))

		admin := CreateSwitch(adminPrivKey, "helloworld")
		servs := []string{RandomSlicePick(services)}
		dept := lib.SERVICE.ServiceDept(servs[0])
		_, _, err = app.AdminManager().RegisterAdmin(
			dept,
			servs,
			admin.NodeInfo().Other[0],
			recvr,
			sendr,
		)
		if err != nil {
			Exit(err.Error())
		}
	*/

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

		// Generate txstr
		txstr := GenerateTxstr(user.NodeInfo().PubKey)

		// Send request
		var req = types.ToRequestAppendTx([]byte(txstr))
		err := types.WriteMessage(req, conn)
		if err != nil {
			t.Fatal(err.Error())
		}

		// Sometimes send flush messages
		if iter%123 == 0 {
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
	symbols = "-.,?!/"
)

var services = lib.SERVICE.GetServices()

func RandomStringPick(str string) string {
	return string(str[rand.Intn(len(str))])
}

func RandomStringPicks(str string, n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		buf.WriteString(RandomStringPick(str))
	}
	return buf.String()
}

func RandomSlicePick(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

func RandOption(serviceName string) string {
	options := lib.SERVICE.FieldOpts(serviceName).GetOptions()
	if options == nil {
		return ""
	}
	return RandomSlicePick(options)
}

func GenerateAddress() string {
	return RandomStringPicks(numbers, 2) + " " + RandomStringPicks(letters, 6) + " st."
}

func GenerateDescription(length int) string {
	var buf bytes.Buffer
	for i := 0; i < length; i++ {
		buf.WriteString(RandomStringPick(letters))
		if i%17 == 0 {
			buf.WriteString(RandomStringPick(symbols))
		} else if i%23 == 0 {
			buf.WriteString(RandomStringPick(numbers))
		}
	}
	return buf.String()
}

func GenerateTxstr(pubKey PubKeyEd25519) string {
	serviceName := RandomSlicePick(services)
	service := lib.SERVICE.WriteField(serviceName, "service")
	address := lib.SERVICE.WriteField(GenerateAddress(), "address")
	description := lib.SERVICE.WriteField(GenerateDescription(60), "description")
	specfield := lib.SERVICE.WriteSpecField(RandOption(serviceName), serviceName)
	pubkey := util.WritePubKeyString(util.PubKeyToString(pubKey))
	return service + address + description + specfield + pubkey
}

// example
// "service {street light out}address {25 Howard St.}description {the light is out.}completely out? {yes}pubkey {19e28ed0accfa56c9f457c943fb8c0fd324d4600429cb22fbfd5cd303d95687e}"
