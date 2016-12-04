package main

import (
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/zballs/comit/actions"
	"github.com/zballs/comit/app"
	"github.com/zballs/comit/server"
	"github.com/zballs/comit/web"
	"net/http"
	"reflect"
	"strings"
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

	// Create comit app
	_app := app.NewApp(cli)

	// If genesis file was specified, set key-value options
	if *genFilePath != "" {
		kvz := loadGenesis(*genFilePath)
		for _, kv := range kvz {
			log := _app.SetOption(kv.Key, kv.Value)
			fmt.Println(Fmt("Set: %v=%v. Log: %v", kv.Key, kv.Value, log))
		}
	}

	// Set State filters
	/*
		filters := append(_app.Issues(), "resolved")
		_app.SetFilters(filters)
	*/

	// Start the listener
	_, err = server.NewSocketServer(*addrPtr, _app)
	if err != nil {
		Exit("create listener: " + err.Error())
	}

	web.RegisterTemplates(
		"account.html",
		"forms.html",
		"network.html",
		"admin.html",
		"index.html",
		"form.html",
	)

	web.CreatePages(
		"account",
		"forms",
		"network",
		"admin",
		"index",
		"form",
	)

	// Create action manager
	addr := strings.Split(*addrPtr, "//")[1]
	am := actions.CreateActionManager(addr)

	js := web.JustFiles{http.Dir("static/")}

	http.HandleFunc("/index", web.TemplateHandler("index.html"))
	http.HandleFunc("/form", web.TemplateHandler("form.html"))

	http.HandleFunc("/account", web.TemplateHandler("account.html"))
	http.HandleFunc("/create-account", am.CreateAccount)
	http.HandleFunc("/remove-account", am.RemoveAccount)

	http.HandleFunc("/network", web.TemplateHandler("network.html"))
	http.HandleFunc("/connect", am.Connect)
	http.HandleFunc("/issues", am.SendIssues)
	http.HandleFunc("/submit-form", am.SubmitForm)
	http.HandleFunc("/resolve-form", am.ResolveForm)
	http.HandleFunc("/update-feed", am.UpdateFeed)

	http.HandleFunc("/forms", web.TemplateHandler("forms.html"))
	http.HandleFunc("/find-form", am.FindForm)

	/*
		http.HandleFunc("/search_forms", am.SearchForms)

		http.HandleFunc("/check_messages", am.CheckMessages)
		http.HandleFunc("/send_message", am.SendMessage)

		http.HandleFunc("/admin", web.TemplateHandler("admin.html"))
		http.HandleFunc("/create_admin", am.CreateAdmin)
		http.HandleFunc("/remove_admin", am.RemoveAdmin)
	*/

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
