package test

import (
	"github.com/zballs/3ii/types"
	"sync"
)

type MutexStruct struct {
	data map[string][]byte
	sync.Mutex
}

func (m MutexStruct) Set(key string, value []byte) {
	m.Lock()
	m.data[key] = value
	m.Unlock()
}

func (m MutexStruct) Get(key string) []byte {
	return m.data[key]
}

type GateStruct struct {
	data map[string][]byte
	gate types.Gate
}

func (g GateStruct) Set(key string, value []byte) {
	g.Access()
	g.data[key] = value
	g.Restore()
}

func (g GateStruct) Get(key string) []byte {
	return g.data[key]
}

func TestMutextStruct() {}

func TestGateStruct() {}
