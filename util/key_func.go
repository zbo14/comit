package util

import (
	"fmt"
	. "github.com/tendermint/go-crypto"
	re "regexp"
)

// Privkey funcs

func ReadPrivKeyEd25519(tx []byte) (privkey PrivKeyEd25519) {
	bytes := re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
	copy(privkey[:], bytes)
	return
}

func ReadPrivKeyBytes(tx []byte) []byte {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
}

func WritePrivKeyBytes(privKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("PrivKeyEd25519{%v}", string(privKeyBytes)))
}

func WritePrivKeyToString(privkey PrivKeyEd25519) string {
	return fmt.Sprintf("PrivKeyEd25519{%v}", string(privkey[:]))
}

func WritePrivKeyToBytes(privkey PrivKeyEd25519) []byte {
	return []byte(WritePrivKeyToString(privkey))
}

func ReadPrivKeyString(tx []byte) string {
	return string(ReadPrivKeyBytes(tx))
}

// Pubkey funcs

func ReadPubKeyEd25519(tx []byte) (pubkey PubKeyEd25519) {
	bytes := re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
	copy(pubkey[:], bytes)
	return
}

func ReadPubKeyBytes(tx []byte) []byte {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
}

func WritePubKeyToString(pubkey PubKeyEd25519) string {
	return fmt.Sprintf("PubKeyEd25519{%v}", string(pubkey[:]))
}

func WritePubKeyToBytes(pubkey PubKeyEd25519) []byte {
	return []byte(WritePubKeyToString(pubkey))
}

func ReadPubKeyString(tx []byte) string {
	return string(ReadPubKeyBytes(tx))
}
