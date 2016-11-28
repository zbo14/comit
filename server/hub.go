package server

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/tendermint/go-wire"
	. "github.com/zballs/comit/util"
	"net"
)

// Client

type Client struct {
	net.Conn
	address []byte
}

func NewClient(conn net.Conn, addr []byte) *Client {
	return &Client{
		Conn:    conn,
		address: addr,
	}
}

// Hub

type Hub struct {
	clients  map[string]net.Conn
	updates  chan []byte
	register chan *Client
}

func NewHub() *Hub {
	return &Hub{

		clients:  make(map[string]net.Conn),
		updates:  make(chan []byte),
		register: make(chan *Client),
	}
}

func (hub *Hub) Register(cli *Client) error {

	select {

	case hub.register <- cli:
		return nil

	default:
		return errors.New("Error: failed to register client")

	}
}

func (hub *Hub) SendUpdate(update []byte) error {

	select {

	case hub.updates <- update:
		fmt.Println("Received update...")
		return nil

	default:
		return errors.New("Error: failed to send update")
	}
}

func (hub *Hub) Run() {

	var bufWriter *bufio.Writer
	var n int
	var err error

	for {

		select {

		case cli := <-hub.register:

			addrString := BytesToHexString(cli.address)

			if _, exists := hub.clients[addrString]; exists {
				continue
			}

			hub.clients[addrString] = cli.Conn
			fmt.Println("Registered client...")

		case update := <-hub.updates:

			for addrString, conn := range hub.clients {

				bufWriter = bufio.NewWriter(conn)
				wire.WriteByteSlice(update, bufWriter, &n, &err)

				if err != nil {
					fmt.Println("Error: could not send update to client")
					delete(hub.clients, addrString)
					continue
				}

				bufWriter.Flush()
				fmt.Printf("%X\n", update)
			}
		}
	}
}

/*

func (hub *Hub) Run() {
	go hub.updatesRoutine()
	// go hub.messagesRoutine()
}

func (hub *Hub) messagesRoutine() {

	for {

		select {

		case message := <-hub.messages:

			fmt.Println("Received message...")
			addrString := BytesToHexString(message.recipient)

			if _, ok := hub.inboxes[addrString]; !ok {
				fmt.Println("Error: could not find recipient inbox")
				continue
			}

			hub.inboxes[addrString] <- message

		case addr := <-hub.newInbox:

			addrString := BytesToHexString(addr)

			if _, exists := hub.inboxes[addrString]; exists {
				continue
			}

			hub.inboxes[addrString] = make(chan Message)
			fmt.Println("Created inbox...")
		}
	}
}

func (hub *Hub) NewInbox(addr []byte) error {

	select {

	case hub.newInbox <- addr:
		return nil

	default:
		return errors.New("Error: failed to open new inbox")
	}
}

func (hub *Hub) SendMessage(message Message) error {

	select {

	case hub.messages <- message:
		return nil

	default:
		return errors.New("Error: failed to send message")

	}
}

func (hub *Hub) CheckMessages(cli *Client) {

	addrString := BytesToHexString(cli.address)

	if _, ok := hub.inboxes[addrString]; !ok {
		return
	}

	var n int
	var err error
	bufWriter := bufio.NewWriter(cli.Conn)

	for message := range hub.inboxes[addrString] {

		bz := wire.BinaryBytes(message)
		wire.WriteByteSlice(bz, bufWriter, &n, &err)

		if err != nil {
			fmt.Println(err.Error())
		}
	}

	bufWriter.Flush()
}
*/
