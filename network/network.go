package network

import (
	cfg "github.com/tendermint/go-config"
	. "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-logger"
	. "github.com/tendermint/go-p2p"
	"net"
	"time"
)

const (
	// Switch config keys
	configKeyDialTimeoutSeconds      = "dial_timeout_seconds"
	configKeyHandshakeTimeoutSeconds = "handshake_timeout_seconds"
	configKeyMaxNumPeers             = "max_num_peers"
	configKeyAuthEnc                 = "authenticated_encryption"

	// MConnection config keys
	configKeySendRate = "send_rate"
	configKeyRecvRate = "recv_rate"

	// Fuzz params
	configFuzzEnable               = "fuzz_enable" // use the fuzz wrapped conn
	configFuzzActive               = "fuzz_active" // toggle fuzzing
	configFuzzMode                 = "fuzz_mode"   // eg. drop, delay
	configFuzzMaxDelayMilliseconds = "fuzz_max_delay_milliseconds"
	configFuzzProbDropRW           = "fuzz_prob_drop_rw"
	configFuzzProbDropConn         = "fuzz_prob_drop_conn"
	configFuzzProbSleep            = "fuzz_prob_sleep"

	// Timeout getMsg
	getMsgTimeout = 200
)

func setConfigDefaults(config cfg.Config) {
	// Switch default config
	config.SetDefault(configKeyDialTimeoutSeconds, 3)
	config.SetDefault(configKeyHandshakeTimeoutSeconds, 20)
	config.SetDefault(configKeyMaxNumPeers, 50)
	config.SetDefault(configKeyAuthEnc, true)

	// MConnection default config
	config.SetDefault(configKeySendRate, 512000) // 500KB/s
	config.SetDefault(configKeyRecvRate, 512000) // 500KB/s

	// Fuzz defaults
	config.SetDefault(configFuzzEnable, false)
	config.SetDefault(configFuzzActive, false)
	config.SetDefault(configFuzzMode, FuzzModeDrop)
	config.SetDefault(configFuzzMaxDelayMilliseconds, 3000)
	config.SetDefault(configFuzzProbDropRW, 0.2)
	config.SetDefault(configFuzzProbDropConn, 0.00)
	config.SetDefault(configFuzzProbSleep, 0.00)
}

var config cfg.Config
var Log = logger.New("module", "p2p")

func init() {
	config = cfg.NewMapConfig(nil)
	setConfigDefaults(config)
}

//===================================================//

// Reactor

type PeerMessage struct {
	PeerKey string
	Bytes   []byte
}

type MyReactor struct {
	BaseReactor
	channels    chan []*ChannelDescriptor
	peers       chan map[string]*Peer
	msgsRecv    map[byte]chan PeerMessage
	logMessages bool
}

func NewReactor(chs []*ChannelDescriptor, logMessages bool) *MyReactor {
	peers := make(chan map[string]*Peer, 1)
	channels := make(chan []*ChannelDescriptor, 1)
	msgsRecv := make(map[byte]chan PeerMessage)
	for _, ch := range chs {
		msgsRecv[ch.ID] = make(chan PeerMessage)
	}
	go func() {
		peers <- map[string]*Peer{}
		channels <- chs
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
		msg_chan := reactor.msgsRecv[chID]
		done := make(chan struct{}, 1)
		go func() {
			msg_chan <- PeerMessage{peer.Key, msgBytes}
			done <- struct{}{}
		}()
		select {
		case <-done:
			return
		}
	}
}

func (reactor *MyReactor) GetMsg(chID byte) PeerMessage {
	msg_chan := reactor.msgsRecv[chID]
	move_on := make(chan struct{}, 1)
	go func() {
		time.Sleep(getMsgTimeout)
		move_on <- struct{}{}
	}()
	select {
	case msg := <-msg_chan:
		return msg
	case <-move_on:
		return PeerMessage{}
	}
}

//======================================================================================//

// Channels

var FeedChannelIDs = map[string]byte{
	"general":        byte(0x00),
	"infrastructure": byte(0x01),
	"sanitation":     byte(0x02),
}

var AdminChannelIDs = map[string]byte{ // by service type or something else?
	"street light out":             byte(0x10),
	"pothole in street":            byte(0x11),
	"rodent baiting/rat complaint": byte(0x12),
	"tree trim":                    byte(0x13),
	"garbage cart black maintenance/replacement": byte(0x14),
}

var recvrListenerAddr = "127.0.0.1:22222"

func RecvrListenerAddr() string {
	return recvrListenerAddr
}

// Switches

func CreateChannelDescriptors(channelIDs []byte) []*ChannelDescriptor {
	chs := make([]*ChannelDescriptor, len(channelIDs))
	for idx, _ := range chs {
		chs[idx] = &ChannelDescriptor{
			ID:       channelIDs[idx],
			Priority: 10,
		}
	}
	return chs
}

func CreateSwitch(privkey PrivKeyEd25519, passphrase string) (sw *Switch) {
	sw = NewSwitch(config)
	sw.SetNodeInfo(&NodeInfo{PubKey: privkey.PubKey().(PubKeyEd25519),
		Network: "testing",
		Version: "311.311.311",
		Other:   []string{passphrase},
	})
	sw.SetNodePrivKey(privkey)
	return
}

func AddListener(sw *Switch, lAddr string) {
	l := NewDefaultListener("tcp", lAddr, false)
	sw.AddListener(l)
}

func AddReactor(sw *Switch, mapChannelIDs map[string]byte, name string) {
	var channelIDs []byte
	for _, chID := range mapChannelIDs {
		channelIDs = append(channelIDs, chID)
	}
	sw.AddReactor(name, NewReactor(CreateChannelDescriptors(channelIDs), true))
}

func Connect2Switches(sw1 *Switch, sw2 *Switch) {
	c1, c2 := net.Pipe()
	go sw1.AddPeerWithConnection(c1, false) // AddPeer is blocking, requires handshake.
	go sw2.AddPeerWithConnection(c2, true)
	time.Sleep(100 * time.Millisecond * time.Duration(4))
}

func DialPeerWithAddr(sw *Switch, lAddr string) (*Peer, error) {
	return sw.DialPeerWithAddress(NewNetAddressString(lAddr))
}
