package network

import (
	"fmt"
	. "github.com/tendermint/go-common"
	cfg "github.com/tendermint/go-config"
	"github.com/tendermint/go-logger"
	. "github.com/tendermint/go-p2p"
	"sync"
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

var Config cfg.Config
var Log = logger.New("module", "p2p")

func init() {
	Config = cfg.NewMapConfig(nil)
	setConfigDefaults(Config)
}

//===================================================//

func waitGroup1() *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return wg
}

type PeerMessage struct {
	PeerKey string
	Bytes   []byte
	Counter int
}

type MyReactor struct {
	BaseReactor

	mtx *sync.Mutex
	wgs map[byte]*sync.WaitGroup

	channels     []*ChannelDescriptor
	peersAdded   []*Peer
	peersRemoved []*Peer
	logMessages  bool
	msgCounters  map[byte]int //counter for each channel
	msgsReceived map[byte][]PeerMessage
}

func NewReactor(channels []*ChannelDescriptor, logMessages bool) *MyReactor {
	mtx := &sync.Mutex{}
	wgs := make(map[byte]*sync.WaitGroup)
	for _, ch := range channels {
		wgs[ch.ID] = waitGroup1()
	}

	mr := &MyReactor{
		mtx:          mtx,
		wgs:          wgs,
		channels:     channels,
		logMessages:  logMessages,
		msgCounters:  make(map[byte]int),
		msgsReceived: make(map[byte][]PeerMessage),
	}
	mr.BaseReactor = *NewBaseReactor(Log, "MyReactor", mr)
	return mr
}

func (mr *MyReactor) GetChannels() []*ChannelDescriptor {
	return mr.channels
}

func (mr *MyReactor) AddPeer(peer *Peer) {
	mr.mtx.Lock()
	defer mr.mtx.Unlock()
	mr.peersAdded = append(mr.peersAdded, peer)
}

func (mr *MyReactor) RemovePeer(peer *Peer, reason interface{}) {
	mr.mtx.Lock()
	defer mr.mtx.Unlock()
	mr.peersRemoved = append(mr.peersRemoved, peer)
}

func (mr *MyReactor) Receive(chID byte, peer *Peer, msgBytes []byte) {
	if mr.logMessages {
		mr.mtx.Lock()
		defer mr.mtx.Unlock()
		fmt.Printf("Received: %X, %X\n", chID, msgBytes)
		counter := mr.msgCounters[chID]
		peer_msg := PeerMessage{peer.Key, msgBytes, counter}
		mr.msgsReceived[chID] = append(mr.msgsReceived[chID], peer_msg)
		if counter == 0 {
			wg := mr.wgs[chID]
			wg.Done()
		}
		mr.msgCounters[chID] = counter + 1
	}
}

func (mr *MyReactor) getMsgs(chID byte) []PeerMessage {
	mr.mtx.Lock()
	defer mr.mtx.Unlock()
	return mr.msgsReceived[chID]
}

func (mr *MyReactor) GetLatestMsg(chID byte) PeerMessage {
	msgs := mr.getMsgs(chID)
	if len(msgs) == 0 {
		wg := mr.wgs[chID]
		// fmt.Println("waiting for message...")
		wg.Wait()
		delete(mr.wgs, chID)
		msgs = mr.getMsgs(chID)
	}
	latest := len(msgs) - 1
	return msgs[latest]
}

//======================================================================================//

// Create Channel Descriptors

func CreateChDescs(mapChannelIDs map[string]byte) []*ChannelDescriptor {
	chDescs := make([]*ChannelDescriptor, len(mapChannelIDs))
	idx := 0
	for _, ch := range mapChannelIDs {
		chDescs[idx] = &ChannelDescriptor{
			ID:       ch,
			Priority: 1,
		}
		idx++
	}
	return chDescs
}

/*
// Service Feed
var ServiceChannelIDs = map[string]byte{
	"street light out":             byte(0x11),
	"pothole in street":            byte(0x12),
	"rodent baiting/rat complaint": byte(0x13),
	"tree trim":                    byte(0x14),
	"garbage cart black maintenance/replacement": byte(0x15),
}
var ServiceChannelDescs = CreateChDescs(ServiceChannelIDs)
var ServiceFeed = NewReactor(ServiceChannelDescs, true)

func ServiceChannelID(service string) uint8 {
	return ServiceChannelIDs[service]
}
*/

// Dept Feed
var DeptChannelIDs = map[string]byte{
	"infrastructure": byte(0x01),
	"sanitation":     byte(0x02),
}
var DeptChannelDescs = CreateChDescs(DeptChannelIDs)
var DeptFeed = NewReactor(DeptChannelDescs, true)

func DeptChannelID(dept string) uint8 {
	return DeptChannelIDs[dept]
}
