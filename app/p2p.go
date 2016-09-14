package app

import (
	// "fmt"
	cfg "github.com/tendermint/go-config"
	// . "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-logger"
	. "github.com/tendermint/go-p2p"
	"net"
)

var Log = logger.New("module", "p2p")
var config = cfg.NewMapConfig(nil)

//===================================================//

type PeerMessage struct {
	PeerKey string
	Bytes   []byte
}

type MyReactor struct {
	BaseReactor
	channels    chan []*ChannelDescriptor
	peers       chan map[string]*Peer
	msgsRecv    chan map[byte][]PeerMessage
	logMessages bool
}

func NewReactor(chs []*ChannelDescriptor, logMessages bool) *MyReactor {
	peers := make(chan map[string]*Peer, 1)
	channels := make(chan []*ChannelDescriptor, 1)
	msgsRecv := make(chan map[byte][]PeerMessage, 1)
	go func() {
		peers <- map[string]*Peer{}
		channels <- chs
		msgsRecv <- map[byte][]PeerMessage{}
	}()
	reactor := &MyReactor{
		channels:    channels,
		peers:       peers,
		msgsRecv:    msgsRecv,
		logMessages: logMessages,
	}
	reactor.BaseReactor = *NewBaseReactor(Log, "MyReactor", reactor)
	return reactor
}

func (reactor *MyReactor) GetChannels() []*ChannelDescriptor {
	channels := <-reactor.channels
	done := make(chan struct{}, 1)
	go func() {
		reactor.channels <- channels
		done <- struct{}{}
	}()
	select {
	case <-done:
		return channels
	}
}

func (reactor *MyReactor) AddPeer(peer *Peer) {
	peers := <-reactor.peers
	if peers[peer.Key] == nil {
		peers[peer.Key] = peer
		done := make(chan struct{}, 1)
		go func() {
			reactor.peers <- peers
			done <- struct{}{}
		}()
		select {
		case <-done:
			return
		}
	}
}

func (reactor *MyReactor) RemovePeer(peer *Peer, reason interface{}) {
	peers := <-reactor.peers
	if peers[peer.Key] != nil {
		delete(peers, peer.Key)
		done := make(chan struct{}, 1)
		go func() {
			reactor.peers <- peers
			done <- struct{}{}
		}()
		select {
		case <-done:
			return
		}
	}
}

func (reactor *MyReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {
	if reactor.logMessages {
		msgs := <-reactor.msgsRecv
		msgs[chID] = append(msgs[chID], PeerMessage{peer.Key, msgBytes})
		done := make(chan struct{}, 1)
		go func() {
			reactor.msgsRecv <- msgs
			done <- struct{}{}
		}()
		select {
		case <-done:
			return
		}
	}
}

func (reactor *MyReactor) getMsgs(chID byte) []PeerMessage {
	msgs := <-reactor.msgsRecv
	done := make(chan struct{}, 1)
	go func() {
		reactor.msgsRecv <- msgs
		done <- struct{}{}
	}()
	select {
	case <-done:
		return msgs[chID]
	}
}

//======================================================================================//

// Create switch pair

func initSwitchFunc(i int, sw *Switch) *Switch {
	sw.AddReactor(
		"feed",
		NewReactor(
			[]*ChannelDescriptor{
				&ChannelDescriptor{
					ID:       byte(0x00),
					Priority: 1,
				},
			},
			true))
	return sw
}

func CreateSwitchPair() (*Switch, *Switch) {
	switches := MakeConnectedSwitches(2, initSwitchFunc, net.Pipe)
	return switches[0], switches[1]
}
