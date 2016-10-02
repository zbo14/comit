package network

import (
	cfg "github.com/tendermint/go-config"
	. "github.com/tendermint/go-crypto"
	"github.com/tendermint/go-logger"
	. "github.com/tendermint/go-p2p"
	"github.com/zballs/3ii/types"
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

	// Timeouts
	getMsgTimeout  = 200
	getPortTimeout = 400
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
	channels    []*ChannelDescriptor
	peers       map[string]*Peer
	msgsRecv    map[byte]chan PeerMessage
	logMessages bool
	BaseReactor
	types.Gate
}

func NewReactor(chs []*ChannelDescriptor, logMessages bool) *MyReactor {
	peers := make(map[string]*Peer)
	channels := make([]*ChannelDescriptor, 0)
	msgsRecv := make(map[byte]chan PeerMessage)
	for _, ch := range chs {
		msgsRecv[ch.ID] = make(chan PeerMessage)
	}
	mr := &MyReactor{
		channels:    channels,
		peers:       peers,
		msgsRecv:    msgsRecv,
		logMessages: logMessages,
	}
	mr.BaseReactor = *NewBaseReactor(Log, "MyReactor", mr)
	return mr
}

func (mr *MyReactor) GetChannels() []*ChannelDescriptor {
	return mr.channels
}

func (mr *MyReactor) AddPeer(peer *Peer) {
	mr.Enter()
	if mr.peers[peer.Key] == nil {
		mr.peers[peer.Key] = peer
	}
	mr.Leave()
}

func (mr *MyReactor) RemovePeer(peer *Peer, reason interface{}) {
	mr.Enter()
	if mr.peers[peer.Key] != nil {
		delete(mr.peers, peer.Key)
	}
	mr.Leave()
}

func (mr *MyReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {
	if mr.logMessages {
		done := types.MakeNilChan()
		go func() {
			mr.msgsRecv[chID] <- PeerMessage{
				peer.Key,
				msgBytes,
			}
			done.Send()
		}()
		select {
		case <-done:
			return
		}
	}
}

func (mr *MyReactor) GetMsg(chID byte) PeerMessage {
	move_on := types.MakeNilChan()
	go func() {
		time.Sleep(getMsgTimeout)
		move_on.Send()
	}()
	select {
	case msg := <-mr.msgsRecv[chID]:
		return msg
	case <-move_on:
		return PeerMessage{}
	}
}

//======================================================================================//

// Channels

var ServiceChannelIDs = map[string]byte{
	"street light out":             byte(0x10),
	"pothole in street":            byte(0x11),
	"rodent baiting/rat complaint": byte(0x12),
	"tree trim":                    byte(0x13),
	"garbage cart black maintenance/replacement": byte(0x14),
}

var DeptChannelIDs = map[string]byte{
	// "general":        byte(0x00),
	"infrastructure": byte(0x01),
	"sanitation":     byte(0x02),
}

func ServiceChannelID(service string) uint8 {
	return ServiceChannelIDs[service]
}

func DeptChannelID(dept string) uint8 {
	return DeptChannelIDs[dept]
}

// Addresses

const (
	RecvrListenerAddr = "127.0.0.1:22222"
)

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

func CreateSwitch(privkey PrivKeyEd25519) (sw *Switch) {
	sw = NewSwitch(config)
	sw.SetNodeInfo(&NodeInfo{PubKey: privkey.PubKey().(PubKeyEd25519),
		Network: "testing",
		Version: "311.311.311",
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
