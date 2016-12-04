package actions

import (
	"bufio"
	ws "github.com/gorilla/websocket"
	"github.com/tendermint/go-wire"
	"github.com/zballs/comit/forms"
	"io"
	"log"
	"net"
	"time"
)

func ReadForm(r io.Reader, form *forms.Form) error {

	n, err := int(0), error(nil)
	data := wire.ReadByteSlice(r, 0, &n, &err)

	if err != nil {
		return err
	}

	err = wire.ReadBinaryBytes(data, form)
	return err
}

// Client
// Reads from TMSP server conn, writes to ws conn

type Client struct {
	In      net.Conn
	Out     *ws.Conn
	Updates chan *forms.Form
}

func NewClient(in net.Conn, out *ws.Conn) *Client {
	return &Client{
		In:      in,
		Out:     out,
		Updates: make(chan *forms.Form),
	}
}

func (cli *Client) ReadRoutine() {

	form := &forms.Form{}
	bufReader := bufio.NewReader(cli.In)

	for {

		err := ReadForm(bufReader, form)

		if err != nil {

			log.Println(err.Error())

			time.Sleep(time.Second * 20)
			continue
		}

		cli.Updates <- form
	}
}

func (cli *Client) WriteRoutine(issues []string, done chan struct{}) {

	defer cli.Out.Close()
	var match bool

	count := 1

	for {

		form, ok := <-cli.Updates
		if !ok {
			close(done)
			return
		}

		match = false
		for _, issue := range issues {
			if form.Issue == issue {
				match = true
				break
			}
		}

		if !match {
			continue
		}

		w, err := cli.Out.NextWriter(ws.TextMessage)
		if err != nil {
			log.Println(err.Error())
			close(done)
			return
		}

		msg := form.Summary("feed", count)

		count++

		w.Write([]byte(msg))

		if len(cli.Updates) > 0 {
			// process queued forms
			for form := range cli.Updates {

				match = false
				for _, issue := range issues {
					if form.Issue == issue {
						match = true
						break
					}
				}

				if !match {
					continue
				}

				msg = form.Summary("feed", count)

				count++

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

/*
func (cli *Client) WriteMessagesRoutine(done chan struct{}) {

	defer cli.Out.Close()

	for {
		message, ok := <-cli.Messages

		if !ok {
			close(done)
			return
		}

		w, err := cli.Out.NextWriter(ws.TextMessage)
		if err != nil {
			log.Println(err.Error())
			close(done)
			return
		}
		messageBytes := message.Format()
		w.Write(messageBytes)

		if len(cli.Messages) > 0 {
			for queued := range cli.Messages {
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
*/
