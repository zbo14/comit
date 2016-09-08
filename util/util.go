package util

import (
	"crypto/sha256"
	rand "math/rand"
)

type Privkey [64]byte
type Pubkey [32]byte

func RandBytes(num int) []byte {
	bytes := make([]byte, num)
	for n := 0; n < num; n++ {
		bytes[n] = byte(rand.Intn(256))
	}
	return bytes
}

func GeneratePrivkey(tx []byte) (privkey Privkey) {
	priv := sha256.Sum256(append(tx, RandBytes(len(privkey))[:]...))
	copy(privkey[:], priv[:])
	return
}

func GeneratePubkey(tx []byte) (pubkey Pubkey) {
	pub := sha256.Sum256(append(tx, RandBytes(len(pubkey))[:]...))
	copy(pubkey[:], pub[:])
	return
}
