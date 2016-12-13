package manager

/*
import (
	"bufio"
	"encoding/json"
	ws "github.com/gorilla/websocket"
	"github.com/tendermint/go-wire"
	. "github.com/zballs/comit/types"
	"io"
	"log"
	"net"
	"time"
)

func ReadForm(r io.Reader, form *Form) error {

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
	Updates chan *Form
}

func NewClient(in net.Conn, out *ws.Conn) *Client {
	return &Client{
		In:      in,
		Out:     out,
		Updates: make(chan *Form),
	}
}

func (cli *Client) ReadRoutine() {

	form := &Form{}
	bufReader := bufio.NewReader(cli.In)

	for {

		err := ReadForm(bufReader, form)

		if err != nil {

			log.Println(err.Error())

			time.Sleep(time.Second * 5)
			continue
		}

		log.Println("Reading...")

		cli.Updates <- form
	}
}

func (cli *Client) WriteRoutine(issue string, done chan struct{}) {

	count := 1

	for {

		form, ok := <-cli.Updates

		if !ok {
			close(done)
			return
		}

		if form.Issue != issue {
			continue
		}

		w, err := cli.Out.NextWriter(ws.TextMessage)

		if err != nil {

			log.Println(err.Error())

			close(done)
			return
		}

		data, err := json.Marshal(form)

		if err != nil {
			panic(err)
		}

		w.Write(data)

		count++

		if len(cli.Updates) > 0 {
			// process queued forms
			for form := range cli.Updates {

				if form.Issue != issue {
					continue
				}

				data, _ = json.Marshal(form)

				w.Write(data)

				count++
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
