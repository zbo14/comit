package actions

import (
	"errors"
	. "github.com/zballs/comit/util"
	"log"
	// "time"
	// . "github.com/tendermint/go-common"
	"sync"
)

type Hub struct {
	sync.Mutex

	// updates
	updates chan []byte
	clients map[string]*Client

	// registration
	register   chan *Client
	unregister chan *Client

	// messages
	messages chan *Message
	inboxes  map[string]chan *Message
}

func NewHub() *Hub {
	return &Hub{
		updates:    make(chan []byte),
		clients:    make(map[string]*Client),
		messages:   make(chan *Message),
		inboxes:    make(map[string]chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (hub *Hub) Register(cli *Client) error {
	select {
	case hub.register <- cli:
		return nil
	default:
		return errors.New("Failed to register client")
	}
}

func (hub *Hub) Unregister(cli *Client) error {
	select {
	case hub.unregister <- cli:
		return nil
	default:
		return errors.New("Failed to unregister client")
	}
}

func (hub *Hub) SendUpdate(update []byte) error {
	select {
	case hub.updates <- update:
		return nil
	default:
		return errors.New("Failed to send update")
	}
}

func (hub *Hub) sendMessage(message *Message) error {
	select {
	case hub.messages <- message:
		return nil
	default:
		return errors.New("Failed to send message")
	}
}

/*
func (hub *Hub) recvMessage(pubKeyString string, message *Message) bool {
	timeout := make(chan *struct{})
	go func() {
		time.Sleep(waitTime)
		close(timeout)
	}()
	select {
	case hub.inboxes[pubKeyString] <- message:
		return true
	case <-timeout:
		return false
	}
}
*/

func (hub *Hub) GetMessages(cli *Client) {
	pubKeyString := BytesToHexString(cli.pubKey[:])
	if _, ok := hub.inboxes[pubKeyString]; !ok {
		// User does not have inbox
		// Create new inbox
		log.Printf("Creating inbox for %v", pubKeyString)
		hub.inboxes[pubKeyString] = make(chan *Message)
	}

	for message := range hub.inboxes[pubKeyString] {
		//log.Println(message)
		cli.messages <- message

		/*
			select {
			case cli.messages <- message:
				continue
			default:
				log.Println("Could not get a message")
			}
		*/
	}
}

// explicit unregister necessary?

func (hub *Hub) Start() {
	for {
		select {
		case cli := <-hub.register:

			pubKeyString := BytesToHexString(cli.pubKey[:])

			if _, exists := hub.clients[pubKeyString]; exists {
				continue
			}

			log.Println("Registered client...")
			hub.clients[pubKeyString] = cli

		case update := <-hub.updates:

			log.Println("Received update...")

			for pubKeyString, cli := range hub.clients {
				sent := cli.sendUpdate(update)
				if !sent {
					log.Println("Error: could not send update to client")
					delete(hub.clients, pubKeyString)
				}
			}
		case message := <-hub.messages:

			log.Println("Received message...")

			pubKeyString := BytesToHexString(message.recipient)

			if _, ok := hub.inboxes[pubKeyString]; !ok {
				// User does not have inbox
				// Create new inbox and send message
				log.Printf("Creating inbox for %v", pubKeyString)
				hub.inboxes[pubKeyString] = make(chan *Message)
			}
			hub.inboxes[pubKeyString] <- message
		}
	}
}

// Separate routines

func (hub *Hub) Run() {
	go hub.registerRoutine()
	go hub.updatesRoutine()
	go hub.messagesRoutine()
}

func (hub *Hub) registerRoutine() {
	for {
		cli := <-hub.register

		pubKeyString := BytesToHexString(cli.pubKey[:])

		hub.Lock()
		if _, exists := hub.clients[pubKeyString]; exists {
			continue
		}

		log.Println("Registered client...")

		hub.clients[pubKeyString] = cli
		hub.Unlock()
	}
}

func (hub *Hub) updatesRoutine() {
	for {
		update := <-hub.updates

		hub.Lock()
		for pubKeyString, cli := range hub.clients {
			sent := cli.sendUpdate(update)
			if !sent {

				log.Println("Error: could not send update to client")

				delete(hub.clients, pubKeyString)
			}
		}
		hub.Unlock()
	}
}

func (hub *Hub) messagesRoutine() {
	for {
		message := <-hub.messages

		log.Println("Received message...")

		pubKeyString := BytesToHexString(message.recipient)

		if _, ok := hub.inboxes[pubKeyString]; !ok {
			// User does not have inbox
			// Create new inbox and send message
			log.Printf("Creating inbox for %v", pubKeyString)
			hub.inboxes[pubKeyString] = make(chan *Message)
		}
		hub.inboxes[pubKeyString] <- message
	}
}
