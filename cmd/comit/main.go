package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/server"
	"github.com/zballs/comit/app"
	"github.com/zballs/comit/manager"
	. "github.com/zballs/comit/util"
	"net/http"
	"reflect"
)

func main() {
	// App
	tmspPtr := flag.String("tmsp", "tcp://0.0.0.0:46658", "Address for tmsp server to listen")
	cliPtr := flag.String("cli", "local", "Client address, or 'local' for embedded")
	// User
	rpcPtr := flag.String("rpc", "tcp://0.0.0.0:46657", "Address of tendermint core rpc server")
	genFilePath := flag.String("genesis", "genesis.json", "Genesis file, if any")
	flag.Parse()

	// Create app client
	cli, err := app.NewClient(*cliPtr, "socket")
	if err != nil {
		Exit("app client: " + err.Error())
	}

	// Create comit app
	comitApp := app.NewApp(cli)

	// If genesis file was specified, set key-value options
	if *genFilePath != "" {
		kvz := loadGenesis(*genFilePath)
		for _, kv := range kvz {
			log := comitApp.SetOption(kv.Key, kv.Value)
			fmt.Println(Fmt("Set: %v=%v. Log: %v", kv.Key, kv.Value, log))
		}
	}

	// Set state filters
	// Just issues for now // TODO: add location
	comitApp.SetFilters()

	// Start the listener
	_, err = server.NewSocketServer(*tmspPtr, comitApp)
	if err != nil {
		Exit("tmsp server: " + err.Error())
	}

	RegisterTemplates("home.html", "citizen.html")
	CreatePages("home", "citizen")

	// Create request multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("/home", TemplateHandler("home.html"))
	mux.HandleFunc("/citizen", TemplateHandler("citizen.html"))

	// Create proxy manager
	m := manager.CreateManager(*rpcPtr)

	// Add routes to multiplexer
	m.AddRoutes(mux)

	// File server
	js := JustFiles{http.Dir("static/")}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(js)))

	// Start HTTP server with multiplexer
	http.ListenAndServe(":8888", mux)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})
}

//------------------------------------------------//

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
