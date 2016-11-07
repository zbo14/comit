package actions

import (
	ws "github.com/gorilla/websocket"
	"github.com/tendermint/go-wire"
	"github.com/zballs/comit/lib"
	"log"
	"time"
)

const (
	waitTime = 10 * time.Second
)

type Client struct {
	*ws.Conn
	updates chan []byte
	issues  map[string]*struct{}
}

func NewClient(conn *ws.Conn, _issues []string) *Client {
	issues := make(map[string]*struct{})
	for _, i := range _issues {
		issues[i] = nil
	}
	return &Client{
		Conn:    conn,
		updates: make(chan []byte),
		issues:  issues,
	}
}

func (cli *Client) sendUpdate(update []byte) bool {
	timeout := make(chan *struct{})
	go func() {
		time.Sleep(waitTime)
		close(timeout)
	}()
	select {
	case cli.updates <- update:
		return true
	case <-timeout:
		return false
	}
}

func (cli *Client) processUpdate(update []byte) (*lib.Form, error) {
	var form lib.Form
	err := wire.ReadBinaryBytes(update, &form)
	if err != nil {
		return nil, err
	}
	issue := form.Issue
	if _, ok := cli.issues[issue]; !ok {
		return nil, nil
	}
	return &form, nil
}

func (cli *Client) writeRoutine(done chan *struct{}) {
	defer cli.Close()
	for {
		update, ok := <-cli.updates
		if !ok {
			close(done)
			return
		}
		form, err := cli.processUpdate(update)
		if err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
		if form == nil {
			// Not a match
			continue
		}
		w, err := cli.NextWriter(ws.TextMessage)
		if err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
		msg := form.Summary()

		w.Write([]byte(msg))

		if len(cli.updates) > 0 {
			for queued := range cli.updates {
				form, err = cli.processUpdate(queued)
				if err != nil {
					log.Println(err.Error()) //for now
					close(done)
					return
				}
				if form == nil {
					// Not a match
					continue
				}
				msg = form.Summary()
				w.Write([]byte(msg))
			}
		}

		if err := w.Close(); err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
	}
}
