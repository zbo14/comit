package state

import (
	"errors"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	"github.com/zballs/comit/types"
)

const (
	ErrSet         = 11311
	iter    uint64 = 3
	members uint64 = 1000
)

type State struct {
	chainID string
	store   types.Store
	blooms  map[string]*types.BloomFilter
	*types.Cache
}

func NewState(store types.Store) *State {
	s := &State{
		chainID: "",
		store:   store,
	}
	s.SetBloomFilters(
		"resolved",
		"street light out",
		"pothole in street",
		"rodent baiting/rat complaint",
		"tree trim",
		"garbage cart black maintenance/replacement",
	)
	return s
}

func (s *State) SetChainID(chainID string) {
	s.chainID = chainID
}

func (s *State) GetChainID() string {
	if s.chainID == "" {
		PanicSanity("Expected to have set chainID")
	}
	return s.chainID
}

func (s *State) SetBloomFilters(filters ...string) {
	blooms := make(map[string]*types.BloomFilter)
	for _, f := range filters {
		blooms[f] = types.MakeBloomFilter(members)
	}
	s.blooms = blooms
}

func (s *State) Get(key []byte) (value []byte) {
	return s.store.Get(key)
}

func (s *State) Set(key []byte, value []byte) {
	s.store.Set(key, value)
}

func (s *State) GetAccount(addr []byte) *types.Account {
	return GetAccount(s.store, addr)
}

func (s *State) SetAccount(addr []byte, acc *types.Account) {
	SetAccount(s.store, addr, acc)
}

func (s *State) AddToFilter(data []byte, filter string) error {
	bloom := s.blooms[filter]
	if bloom == nil {
		return errors.New(
			Fmt("Cannot find filter %s", filter))
	}
	if bloom.HasMember(data, iter) {
		return errors.New(
			Fmt("%X already in filter %s", data, filter))
	}
	bloom.AddMember(data, iter)
	return nil
}

func (s *State) InFilter(filter string, data []byte) bool {
	bloom := s.blooms[filter]
	if bloom == nil {
		// errors.New(Fmt("Cannot find filter %s", filter))
		return false //for now
	}
	if !bloom.HasMember(data, iter) {
		return false
	}
	return true
}

func (s *State) FilterFunc(filters []string) func(data []byte) bool {
	return func(data []byte) bool {
		for _, f := range filters {
			if has := s.InFilter(f, data); !has {
				return false
			}
		}
		return true
	}
}

func (s *State) CacheWrap() *State {
	cache := types.NewCache(s.store)
	snew := &State{
		chainID: s.chainID,
		store:   cache,
		blooms:  s.blooms,
	}
	snew.Cache = cache
	return snew
}

func (s *State) CacheSync() {
	s.Sync()
}

func AccountKey(addr []byte) []byte {
	return append([]byte("base/a/"), addr...)
}

func GetAccount(store types.Store, addr []byte) *types.Account {
	data := store.Get(AccountKey(addr))
	if len(data) == 0 {
		return nil
	}
	var acc *types.Account
	err := wire.ReadBinaryBytes(data, &acc)
	if err != nil {
		panic(Fmt("Error reading account %X error: %v",
			data, err.Error()))
	}
	return acc
}

func SetAccount(store types.Store, addr []byte, acc *types.Account) {
	accBytes := wire.BinaryBytes(acc)
	store.Set(AccountKey(addr), accBytes)
}
