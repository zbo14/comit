package actions

import (
	"bytes"
	ws "github.com/gorilla/websocket"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/zballs/comit/lib"
	"log"
	"time"
)

const (
	waitTime = 10 * time.Second
)

type Message struct {
	sender    []byte
	recipient []byte
	content   []byte
}

func NewMessage() *Message {
	senderBytes := make([]byte, 32)
	recipientBytes := make([]byte, 32)
	return &Message{
		sender:    senderBytes,
		recipient: recipientBytes,
	}
}

func (m *Message) Format() []byte {
	var buf bytes.Buffer
	buf.WriteString(
		Fmt("<small><strong>from</strong></small> <really-small>%X</really-small><br>", m.sender))
	buf.Write(m.content)
	return buf.Bytes()
}

type Client struct {
	*ws.Conn
	pubKey   crypto.PubKeyEd25519
	updates  chan []byte
	messages chan *Message
}

func NewClient(conn *ws.Conn, pubKey crypto.PubKeyEd25519) *Client {
	return &Client{
		Conn:     conn,
		pubKey:   pubKey,
		updates:  make(chan []byte),
		messages: make(chan *Message),
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

func (cli *Client) processUpdate(update []byte, issues []string) (*lib.Form, error) {
	var form lib.Form
	err := wire.ReadBinaryBytes(update, &form)
	if err != nil {
		return nil, err
	}
	issue := form.Issue
	for _, _issue := range issues {
		if issue == _issue {
			return &form, nil
		}
	}
	return nil, nil
}

func (cli *Client) writeUpdatesRoutine(issues []string, done chan *struct{}) {

	defer cli.Close()

	for {
		update, ok := <-cli.updates
		if !ok {
			close(done)
			return
		}
		form, err := cli.processUpdate(update, issues)
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
		msg := form.Summary("feed", 0)

		w.Write([]byte(msg))

		if len(cli.updates) > 0 {
			for queued := range cli.updates {
				form, err = cli.processUpdate(queued, issues)
				if err != nil {
					log.Println(err.Error()) //for now
					close(done)
					return
				}
				if form == nil {
					// Not a match
					continue
				}
				msg = form.Summary("feed", 0)
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

func (cli *Client) writeMessagesRoutine(done chan *struct{}) {

	defer cli.Close()

	for {
		message, ok := <-cli.messages

		if !ok {
			close(done)
			return
		}

		w, err := cli.NextWriter(ws.TextMessage)
		if err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
		messageBytes := message.Format()
		w.Write(messageBytes)

		if len(cli.messages) > 0 {
			for queued := range cli.messages {
				messageBytes := queued.Format()
				w.Write(messageBytes)
			}
		}

		if err := w.Close(); err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
	}
}
