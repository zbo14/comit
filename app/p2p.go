package app

import (
	cfg "github.com/tendermint/go-config"
	"github.com/tendermint/go-logger"
	. "github.com/tendermint/go-p2p"
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
	name        string
	channels    chan []*ChannelDescriptor
	peers       chan map[string]*Peer
	msgsRecv    chan map[byte][]PeerMessage
	logMessages bool
}

func NewReactor(name string, chs []*ChannelDescriptor, logMessages bool) *MyReactor {
	peers := make(chan map[string]*Peer, 1)
	channels := make(chan []*ChannelDescriptor, 1)
	go func() {
		peers <- map[string]*Peer{}
		channels <- chs
	}()
	reactor := &MyReactor{
		name:        name,
		channels:    channels,
		peers:       peers,
		logMessages: logMessages,
	}
	reactor.BaseReactor = *NewBaseReactor(Log, name, reactor)
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

func CreateChannelDescriptor(id int, priority int) *ChannelDescriptor {
	return &ChannelDescriptor{
		ID:       byte(id),
		Priority: priority,
	}
}

func CreateReactor(name string, logMessages bool, chs ...*ChannelDescriptor) *MyReactor {
	return NewReactor(name, chs, logMessages)
}

func CreateSwitch(config cfg.Config, reactors ...*MyReactor) (sw *Switch) {
	sw = NewSwitch(config)
	for _, r := range reactors {
		sw.AddReactor(r.name, r)
	}
	return
}

//====================================================//

var ch0 = CreateChannelDescriptor(0, 1)
var msgBoard = CreateReactor("msgBoard", true, ch0)
