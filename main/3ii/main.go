package main

import (
	"flag"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/server"
	. "github.com/zballs/3ii/actions"
	. "github.com/zballs/3ii/app"
	. "github.com/zballs/3ii/types"
	"net/http"
)

func main() {

	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	tmspPtr := flag.String("tmsp", "socket", "socket | grpc")
	flag.Parse()

	// Start the listener
	app := NewApp()
	_, err := server.NewServer(*addrPtr, *tmspPtr, app)
	if err != nil {
		Exit(err.Error())
	}

	RegisterTemplates(
		"create_account.html",
		"remove_account.html",
		"submit_form.html",
		"resolve_form.html",
		"find_form.html",
	)

	CreatePages(
		"create_account",
		"remove_account",
		"submit_form",
		"resolve_form",
		"find_form",
	)

	action_listener, err := StartActionListener()
	if err != nil {
		Exit(err.Error())
	}

	// go action_listener.FeedUpdates()

	action_listener.Run(app)

	js := JustFiles{http.Dir("static/")}
	http.Handle("/", action_listener)
	http.HandleFunc("/create_account", CustomHandler("create_account.html"))
	http.HandleFunc("/remove_account", CustomHandler("remove_account.html"))
	http.HandleFunc("/submit_form", CustomHandler("submit_form.html"))
	http.HandleFunc("/resolve_form", CustomHandler("resolve_form.html"))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(js)))
	http.ListenAndServe(":8888", nil)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
