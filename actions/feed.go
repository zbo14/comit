package actions

import (
	"errors"
	"log"
)

type Feed struct {
	updates  chan []byte
	clients  map[*Client]*struct{}
	register chan *Client
}

func NewFeed() *Feed {
	return &Feed{
		updates:  make(chan []byte),
		clients:  make(map[*Client]*struct{}),
		register: make(chan *Client),
	}
}

func (feed *Feed) Register(cli *Client) error {
	select {
	case feed.register <- cli:
		return nil
	default:
		return errors.New("Failed to register client")
	}
}

func (feed *Feed) SendUpdate(update []byte) error {
	select {
	case feed.updates <- update:
		return nil
	default:
		return errors.New("Failed to send update")
	}
}

func (feed *Feed) Start() {
	for {
		select {
		case cli := <-feed.register:

			log.Println("Registered client...")

			if _, ok := feed.clients[cli]; !ok {
				feed.clients[cli] = nil
			}

		case update := <-feed.updates:

			log.Println("Received update...")

			for cli := range feed.clients {
				sent := cli.sendUpdate(update)
				if !sent {
					delete(feed.clients, cli)
				}
			}
		}
	}
}
