package types

import (
	"fmt"
	. "github.com/tendermint/go-common"
	// sync
)

type Cache struct {
	*KVMap
	store Store
	gate  Gate
	// mtx *sync.Mutex
}

func NewCache(store Store) *Cache {
	c := &Cache{
		store: store,
		// mtx:   &sync.Mutex{},
		gate: MakeGate(),
	}
	c.Reset()
	return c
}

func (c *Cache) Reset() {
	c.KVMap = NewKVMap()
}

func (c *Cache) Set(key []byte, value []byte) {
	c.gate.Enter()
	defer c.gate.Leave()
	fmt.Println("Set [Cache]", formatBytes(key, "string"), "=", formatBytes(value, "string"))
	c.KVMap.Set(key, value)
}

func (c *Cache) Get(key []byte) (value []byte) {
	value = c.KVMap.Get(key)
	if value != nil {
		fmt.Println("GET [Cache, hit]", formatBytes(key, "string"), "=", formatBytes(value, "string"))
		return
	}
	value = c.store.Get(key)
	c.Set(key, value)
	fmt.Println("GET [Cache, miss]", formatBytes(key, "string"), "=", formatBytes(value, "string"))
	return
}

func (c *Cache) Sync() {
	c.gate.Enter()
	defer c.gate.Leave()
	for kvn := c.KVList.head; kvn != nil; kvn = kvn.next {
		c.store.Set(kvn.key, kvn.value)
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
