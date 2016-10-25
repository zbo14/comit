package types

import (
	// "encoding/binary"
	"fmt"
)

type BloomFilter struct {
	bytes    []byte
	capacity uint64
	members  uint64
}

// CHALLENGE: make extensible

func MakeBloomFilter(c uint64) *BloomFilter {
	bytes := make([]byte, 6*c/5) // 1.2 bytes per member
	return &BloomFilter{
		bytes:    bytes,
		capacity: c,
		members:  0,
	}
}

// change some uint64's -> uint's

func (bloom *BloomFilter) getIdx(pos uint64) uint64 {
	return uint64(len(bloom.bytes)-1) - (pos / 8)
}

func (bloom *BloomFilter) length() uint64 {
	return uint64(len(bloom.bytes))
}

func (bloom *BloomFilter) size() uint64 {
	return bloom.length() * 8
}

func (bloom *BloomFilter) setBit(pos uint64) {
	idx := bloom.getIdx(pos)
	b := bloom.bytes[idx]
	b |= (1 << (pos % 8))
	bloom.bytes[idx] = b
	// fmt.Printf("%.8b\n", bloom.bytes)
}

func (bloom *BloomFilter) clearBit(pos uint64) {
	idx := bloom.getIdx(pos)
	b := bloom.bytes[idx]
	b &^= (1 << (pos % 8))
	bloom.bytes[idx] = b
	// fmt.Printf("%.8b\n", bloom.bytes)
}

func (bloom *BloomFilter) hasBit(pos uint64) bool {
	idx := bloom.getIdx(pos)
	b := bloom.bytes[idx]
	val := b & (1 << (pos % 8))
	return (val > 0)
}

// Hash functions

const (

	// Murmur
	c1     uint64 = 0xcc9e2d51
	c2     uint64 = 0x1b873593
	n      uint64 = 0xe6546b64
	round4 uint64 = 0xfffffffc
	seed   uint64 = 0x0

	// FNV
	FNV_offset_basis uint64 = 0xcbf29ce484222325
	FNV_prime        uint64 = 0x100000001b3
)

func (bloom *BloomFilter) murmurHash(bytes []byte, seed uint64) uint64 {
	h := seed
	length := uint64(len(bytes))
	roundedEnd := length & round4
	var i uint64
	var k uint64
	for i = 0; i < roundedEnd; i += 4 {
		b0, b1, b2, b3 := bytes[i], bytes[i+1], bytes[i+2], bytes[i+3]
		k := uint64(b0 | (b1 << 8) | (b2 << 16) | (b3 << 24))
		// k, _ := binary.Uvarint(bytes[i : i+4])
		k *= c1
		k = (k << 15) | (k >> 17)
		k *= c2
		h ^= k
		h = (h << 13) | (h >> 19)
		h = h*5 + n
	}
	k = 0
	val := length & 0x03
	if val == 3 {
		k = uint64(bytes[roundedEnd+2] << 16)
	}
	if val >= 2 {
		k |= uint64(bytes[roundedEnd+1] << 8)
	}
	if val >= 1 {
		k |= uint64(bytes[roundedEnd])
		k *= c1
		k = (k << 15) | (k >> 17)
		k *= c2
		h ^= k
	}
	h ^= length
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func (bloom *BloomFilter) fnvHash(bytes []byte) uint64 {
	hash := FNV_offset_basis
	for _, b := range bytes {
		hash *= FNV_prime
		hash |= uint64(b)
	}
	return hash
}

func (bloom *BloomFilter) jenkinsHash(bytes []byte) uint64 {
	var hash uint64
	for _, b := range bytes {
		hash += uint64(b)
		hash += (hash << 10)
		hash ^= (hash >> 6)
	}
	hash += (hash << 3)
	hash ^= (hash >> 11)
	hash += (hash << 15)
	return hash
}

func (bloom *BloomFilter) HasMember(bytes []byte, n uint64) bool {
	hash := bloom.murmurHash(bytes, seed)
	var i uint64
	for i = 0; i < n; i++ {
		hash += bloom.fnvHash(bytes)
		pos := hash % bloom.size()
		if !bloom.hasBit(pos) {
			fmt.Printf("%X not a member\n", bytes)
			return false
		}
	}
	fmt.Printf("%X is a member\n", bytes)
	return true
}

func (bloom *BloomFilter) setBits(bytes []byte, n uint64) {
	hash := bloom.murmurHash(bytes, seed)
	size := bloom.size()
	var i uint64
	for i = 0; i < n; i++ {
		hash += bloom.fnvHash(bytes)
		pos := hash % size
		if !bloom.hasBit(pos) {
			bloom.setBit(pos)
		}
	}
}

func (bloom *BloomFilter) AddMember(bytes []byte, n uint64) {
	if bloom.HasMember(bytes, n) {
		fmt.Printf("%X is already a member\n", bytes)
		return
	}
	bloom.setBits(bytes, n)
	bloom.members++
	fmt.Printf("added member %X\n", bytes)
}

func (bloom *BloomFilter) OverCapacity() bool {
	return bloom.members > bloom.capacity
}
