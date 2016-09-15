package main

import (
	"flag"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/tmsp/server"
	. "github.com/zballs/3ii/app"
	. "github.com/zballs/3ii/types"
	"log"
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
		"find_form.html",
		"resolve_form.html",
		"search_forms.html",
		"feed.html",
	)

	CreatePages(
		"create_account",
		"create_admin",
		"remove_account",
		"remove_admin",
		"submit_form",
		"find_form",
		"resolve_form",
		"search_forms",
		"feed",
	)

	action_listener, err := CreateActionListener()
	if err != nil {
		Exit(err.Error())
	}

	// Recv Feed updates
	go func() {
		if action_listener.Receiver.IsRunning() {
			log.Println(action_listener.Receiver.Peers().Size())
			updates := action_listener.Receiver.Reactor("feed").(*MyReactor).GetMsgs(byte(0x00))
			if len(updates) > 0 {
				update := updates[len(updates)-1]
				action_listener.BroadcastTo("feed", "feed-update", update)
			}
		}
	}()

	action_listener.Run(app)

	js := JustFiles{http.Dir("static/")}
	http.Handle("/", action_listener)
	http.HandleFunc("/create_account", CreateAccountHandler)
	http.HandleFunc("/create_admin", CreateAdminHandler)
	http.HandleFunc("/remove_account", RemoveAccountHandler)
	http.HandleFunc("/remove_admin", RemoveAdminHandler)
	http.HandleFunc("/submit_form", SubmitFormHandler)
	http.HandleFunc("/find_form", FindFormHandler)
	http.HandleFunc("/resolve_form", ResolveFormHandler)
	http.HandleFunc("/search_forms", SearchFormsHandler)
	http.HandleFunc("/feed", FeedHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(js)))
	http.ListenAndServe(":8888", nil)

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
