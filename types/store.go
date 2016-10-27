package types

import (
	. "github.com/zballs/comit/util"
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
