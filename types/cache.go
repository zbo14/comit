package types

import (
	"fmt"
	. "github.com/tendermint/go-common"
	. "github.com/zballs/3ii/util"
	// "sync"
)

type Store interface {
	Set(key, value []byte)
	Get(key []byte) (value []byte)
}

type MemStore struct {
	m map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{
		m: make(map[string][]byte, 0),
	}
}

func (mstore *MemStore) Set(key []byte, value []byte) {
	mstore.m[BytesToHexString(key)] = value
}

func (mstore *MemStore) Get(key []byte) (value []byte) {
	return mstore.m[BytesToHexString(key)]
}

type KVMaps struct {
	nums_keys map[int]string
	keys_nums map[string]int
	keys_vals map[string][]byte
}

type Cache struct {
	KVMaps
	store Store
	gate  Gate
}

// Goroutine safe Cache with store

func NewCache(store Store) *Cache {
	return (&Cache{
		store: store,
		gate:  MakeGate(),
	}).Reset()
}

func (c *Cache) Reset() *Cache {
	c.KVMaps = KVMaps{
		nums_keys: make(map[int]string),
		keys_nums: make(map[string]int),
		keys_vals: make(map[string][]byte),
	}
	return c
}

func (c *Cache) Set(key []byte, value []byte) {
	c.gate.Enter()
	defer c.gate.Leave()
	fmt.Println("Set [Cache]", formatBytes(key, "hex"), "=", formatBytes(value, "string"))
	keystr := BytesToHexString(key)
	c.keys_vals[keystr] = value
	length := len(c.keys_vals)
	keynum := c.keys_nums[keystr]
	if keynum > 0 {
		for n := keynum + 1; n <= length; n++ {
			k := c.nums_keys[n]
			c.keys_nums[k] -= 1
			c.nums_keys[n-1] = k
		}
	}
	c.keys_nums[keystr] = length
	c.nums_keys[length] = keystr
}

func (c *Cache) Get(key []byte) (value []byte) {
	value = c.keys_vals[BytesToHexString(key)]
	if value != nil {
		fmt.Println("GET [Cache, hit]", formatBytes(key, "hex"), "=", formatBytes(value, "string"))
		return
	}
	value = c.store.Get(key)
	c.Set(key, value)
	fmt.Println("GET [Cache, miss]", formatBytes(key, "hex"), "=", formatBytes(value, "string"))
	return
}

func (c *Cache) Sync() {
	c.gate.Enter()
	defer c.gate.Leave()
	for n := 1; n <= len(c.keys_vals); n++ {
		keystr := c.nums_keys[n]
		value := c.keys_vals[keystr]
		c.store.Set([]byte(keystr), value)
	}
	c.Reset()
}

func formatBytes(data []byte, mode string) (str string) {
	if mode == "string" {
		for _, b := range data {
			if 0x21 <= b && b < 0x7F {
				str += Green(string(b))
			} else {
				str += Red("?")
			}
		}
	} else if mode == "hex" {
		str = Blue(Fmt("%X", data))
	}
	return
}
