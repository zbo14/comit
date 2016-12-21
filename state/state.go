package state

import (
	"github.com/pkg/errors"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	"github.com/zballs/comit/types"
	"github.com/zballs/dl_cbf"
)

type State struct {
	chainID string
	filters map[string]dl_cbf.HashTable
	store   types.Store
	*types.Cache
}

func NewState(store types.Store) *State {
	s := &State{
		chainID: "",
		store:   store,
	}
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

func (s *State) SetFilters(names []string) {
	filters := make(map[string]dl_cbf.HashTable)
	for _, name := range names {
		filters[name], _ = dl_cbf.NewHashTable_Default32(10000000)
	}
	s.filters = filters
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

func (s *State) FilterAdd(data []byte, name string) error { //must add
	filter, ok := s.filters[name]
	if !ok {
		return errors.New(Fmt("Failed to find '%s' filter", name))
	}
	_, success := filter.Add(data)
	if !success {
		return errors.New(Fmt("Failed to add data to '%s' filter", name))
	}
	return nil
}

func (s *State) FilterLookup(data []byte, name string) (bool, error) {
	filter, ok := s.filters[name]
	if !ok {
		return false, errors.New(Fmt("Failed to find '%s' filter", name))
	}
	_, found := filter.Lookup(data)
	return found, nil
}

func (s *State) FilterCount(data []byte, name string) (int, error) {
	filter, ok := s.filters[name]
	if !ok {
		return 0, errors.New(Fmt("Failed to find '%s' filter", name))
	}
	_, count := filter.GetCount(data)
	return int(count), nil
}

func (s *State) FilterDelete(data []byte, name string) error { //must delete
	filter, ok := s.filters[name]
	if !ok {
		return errors.New(Fmt("Failed to find '%s' filter", name))
	}
	_, success := filter.Delete(data)
	if !success {
		return errors.New(Fmt("Failed to delete data from '%s' filter", name))
	}
	return nil
}

func (s *State) Filterfunc(name string) func([]byte) bool {
	return func(data []byte) bool {
		has, err := s.FilterLookup(data, name)
		if err != nil {
			panic(err)
		}
		return has
	}
}

func (s *State) Filtersfunc(names []string) func([]byte) bool {
	return func(data []byte) bool {
		for _, name := range names {
			has := s.Filterfunc(name)(data)
			if !has {
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
		filters: s.filters,
		store:   cache,
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
