package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-p2p"
	"github.com/tendermint/tmsp/server"
	"github.com/zballs/comit/actions"
	"github.com/zballs/comit/app"
	ntwk "github.com/zballs/comit/network"
	"github.com/zballs/comit/web"
	"net/http"
	"reflect"
)

func main() {

	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	cliPtr := flag.String("cli", "local", "Client address, or 'local' for embedded")
	networkPtr := flag.String("network", "127.0.0.1:3111", "Feeds address")
	peerPtr := flag.String("peer", "127.0.0.1:3112", "Peer address")
	genFilePath := flag.String("genesis", "genesis.json", "Genesis file, if any")
	flag.Parse()

	// Connect to Client
	cli, err := app.NewClient(*cliPtr, "socket")
	if err != nil {
		Exit("connect to client: " + err.Error())
	}

	// Create comit app
	app_ := app.NewApp(cli)

	// If genesis file was specified, set key-value options
	if *genFilePath != "" {
		kvz := loadGenesis(*genFilePath)
		for _, kv := range kvz {
			log := app_.SetOption(kv.Key, kv.Value)
			fmt.Println(Fmt("Set: %v=%v. Log: %v", kv.Key, kv.Value, log))
		}
	}

	// State filters
	filters := append(app_.Issues(), []string{"resolved", "unresolved"}...)
	app_.SetFilters(filters)

	// Create reactors for network
	issues := app_.CreateIssueReactor()
	admins := app_.CreateAdminReactor()

	// Start the listener
	_, err = server.NewServer(*addrPtr, "socket", app_)
	if err != nil {
		Exit("create listener: " + err.Error())
	}

	// Start the network
	network := p2p.NewSwitch(ntwk.Config)
	network.SetNodeInfo(&p2p.NodeInfo{
		Network: "testing",
		Version: "311.311.311",
	})
	network.SetNodePrivKey(crypto.GenPrivKeyEd25519())
	network.AddReactor("issues", issues)
	network.AddReactor("admins", admins)
	l := p2p.NewDefaultListener("tcp", *networkPtr, false)
	network.AddListener(l)
	network.Start()

	web.RegisterTemplates(
		"account.html",
		"forms.html",
		"network.html",
	)

	web.CreatePages(
		"account",
		"forms",
		"network",
	)

	// Create action manager
	am := actions.CreateActionManager(app_, network, *peerPtr)

	js := web.JustFiles{http.Dir("static/")}

	http.HandleFunc("/account", web.TemplateHandler("account.html"))
	http.HandleFunc("/connect_accout", am.ConnectAccount)
	http.HandleFunc("/create_account", am.CreateAccount)
	http.HandleFunc("/remove_account", am.RemoveAccount)

	http.HandleFunc("/get_issues", am.GetIssues)

	http.HandleFunc("/forms", web.TemplateHandler("forms.html"))
	http.HandleFunc("/submit_form", am.SubmitForm)
	http.HandleFunc("/find_form", am.FindForm)
	http.HandleFunc("/search_forms", am.SearchForms)

	http.HandleFunc("/network", web.TemplateHandler("web.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(js)))
	http.ListenAndServe(":8888", nil)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})
}

//----------------------------------------

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func loadGenesis(filePath string) (kvz []KeyValue) {
	kvz_ := []interface{}{}
	bytes, err := ReadFile(filePath)
	if err != nil {
		Exit("loading genesis file: " + err.Error())
	}
	err = json.Unmarshal(bytes, &kvz_)
	if err != nil {
		Exit("parsing genesis file: " + err.Error())
	}
	if len(kvz_)%2 != 0 {
		Exit("genesis cannot have an odd number of items.  Format = [key1, value1, key2, value2, ...]")
	}
	for i := 0; i < len(kvz_); i += 2 {
		keyIfc := kvz_[i]
		valueIfc := kvz_[i+1]
		var key, value string
		key, ok := keyIfc.(string)
		if !ok {
			Exit(Fmt("genesis had invalid key %v of type %v", keyIfc, reflect.TypeOf(keyIfc)))
		}
		if value_, ok := valueIfc.(string); ok {
			value = value_
		} else {
			valueBytes, err := json.Marshal(valueIfc)
			if err != nil {
				Exit(Fmt("genesis had invalid value %v: %v", value_, err.Error()))
			}
			value = string(valueBytes)
		}
		kvz = append(kvz, KeyValue{key, value})
	}
	return kvz
}

/*

// Create action listener
action_listener, err := actions.CreateActionListener()
if err != nil {
	Exit("action listener: " + err.Error())
}

http.Handle("/", action_listener)
http.HandleFunc("/create_admin", web.TemplateHandler("create_admin.html"))
http.HandleFunc("/remove_account", web.TemplateHandler("remove_account.html"))
http.HandleFunc("/submit_form", web.TemplateHandler("submit_form.html"))
http.HandleFunc("/resolve_form", web.TemplateHandler("resolve_form.html"))
http.HandleFunc("/find_form", web.TemplateHandler("find_form.html"))
http.HandleFunc("/search_forms", web.TemplateHandler("search_forms.html"))
http.HandleFunc("/connect", web.TemplateHandler("connect.html"))

action_listener.Run(app_, network, *peerPtr)

func writeToGenesis(key, value interface{}, filePath string) error {
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteJSON(&key, buf, &n, &err)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	f.Write(buf.Bytes())
	buf = new(bytes.Buffer)
	wire.WriteJSON(&value, buf, &n, &err)
	f.Write(buf.Bytes())
	f.Close()
	return nil
}

// Service Feed
var ServiceChannelIDs = map[string]byte{
	"street light out":             byte(0x11),
	"pothole in street":            byte(0x12),
	"rodent baiting/rat complaint": byte(0x13),
	"tree trim":                    byte(0x14),
	"garbage cart black maintenance/replacement": byte(0x15),
}
var ServiceChannelDescs = CreateChDescs(ServiceChannelIDs)
var ServiceFeed = NewReactor(ServiceChannelDescs, true)

func ServiceChannelID(service string) uint8 {
	return ServiceChannelIDs[service]
}
*/
