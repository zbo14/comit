package main

import (
	"flag"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/server"
	. "github.com/zballs/3ii/app"
	. "github.com/zballs/3ii/types"
	"net/http"
)

func main() {

	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	tmspPtr := flag.String("tmsp", "socket", "socket | grpc")
	flag.Parse()

	// Start the listener
	app := NewApplication()
	_, err := server.NewServer(*addrPtr, *tmspPtr, app)
	if err != nil {
		Exit(err.Error())
	}

	RegisterTemplates(
		"create_account.html",
		"create_admin.html",
		"remove_account.html",
		"remove_admin.html",
		"submit_form.html",
		"query_form.html",
		"resolve_form.html",
	)

	CreatePages(
		"create_account",
		"create_admin",
		"remove_account",
		"remove_admin",
		"submit_form",
		"query_form",
		"resolve_form",
	)

	action_listener, err := CreateActionListener()
	if err != nil {
		Exit(err.Error())
	}

	action_listener.Run(app)
	js := JustFiles{http.Dir("static/")}
	http.Handle("/", action_listener)
	http.HandleFunc("/create_account", CreateAccountHandler)
	http.HandleFunc("/create_admin", CreateAdminHandler)
	http.HandleFunc("/remove_account", RemoveAccountHandler)
	http.HandleFunc("/remove_admin", RemoveAdminHandler)
	http.HandleFunc("/submit_form", SubmitFormHandler)
	http.HandleFunc("/query_form", QueryFormHandler)
	http.HandleFunc("/resolve_form", ResolveFormHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(js)))
	http.ListenAndServe(":8888", nil)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
