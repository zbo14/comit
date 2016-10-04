package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/server"
	"github.com/zballs/3ii/actions"
	"github.com/zballs/3ii/app"
	// "github.com/zballs/3ii/types"
	"github.com/zballs/3ii/web"
	"net/http"
	"reflect"
)

func main() {

	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	cliPtr := flag.String("cli", "local", "Client address, or 'local' for embedded")
	genFilePath := flag.String("genesis", "genesis.json", "Genesis file, if any")
	flag.Parse()

	// Connect to Client
	cli, err := app.NewClient(*cliPtr, "socket")
	if err != nil {
		Exit("connect to client: " + err.Error())
	}

	// Create 3ii app
	app_ := app.NewApp(cli)

	// If genesis file was specified, set key-value options
	if *genFilePath != "" {
		kvz := loadGenesis(*genFilePath)
		for _, kv := range kvz {
			log := app_.SetOption(kv.Key, kv.Value)
			fmt.Println(Fmt("Set: %v=%v. Log: %v", kv.Key, kv.Value, log))
		}
	}

	// Start the listener
	_, err = server.NewServer(*addrPtr, "socket", app_)
	if err != nil {
		Exit("create listener: " + err.Error())
	}

	web.RegisterTemplates(
		"create_account.html",
		"remove_account.html",
		"submit_form.html",
		"resolve_form.html",
		"find_form.html",
	)

	web.CreatePages(
		"create_account",
		"remove_account",
		"submit_form",
		"resolve_form",
		"find_form",
	)

	action_listener, err := actions.StartActionListener()
	if err != nil {
		Exit("action listener: " + err.Error())
	}

	// go action_listener.FeedUpdates()

	action_listener.Run(app_)

	js := web.JustFiles{http.Dir("static/")}
	http.Handle("/", action_listener)
	http.HandleFunc("/create_account", web.CustomHandler("create_account.html"))
	http.HandleFunc("/remove_account", web.CustomHandler("remove_account.html"))
	http.HandleFunc("/submit_form", web.CustomHandler("submit_form.html"))
	http.HandleFunc("/resolve_form", web.CustomHandler("resolve_form.html"))
	http.HandleFunc("/find_form", web.CustomHandler("find_form.html"))
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
