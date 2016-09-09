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

	CreatePages("submit")
	tmpl := RegisterTemplates("submit.html")

	action_listener, err := CreateActionListener()
	if err != nil {
		Exit(err.Error())
	}

	action_listener.SubmitForms(CreateAccountManager(), app)
	js := JustFiles{http.Dir("static/")}
	http.Handle("/submit", SubmitHandler)
	http.Handler("/static/", http.StripPrefix("/static/", http.FileServer(js)))
	http.ListenAndServe(":8888", nil)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
