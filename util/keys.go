package util

import (
	"fmt"
	. "github.com/tendermint/go-crypto"
	re "regexp"
)

// Inscribe, extract privkey

func ExtractPrivKeyEd25519(tx []byte) (privkey PrivKeyEd25519) {
	bytes := re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
	copy(privkey[:], bytes)
	return
}

func ExtractPrivKeyBytes(tx []byte) []byte {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
}

func InscribePrivKeyString(privkey PrivKeyEd25519) string {
	return fmt.Sprintf("PrivKeyEd25519{%X}", privkey[:])
}

func InscribePrivKeyBytes(privkey PrivKeyEd25519) []byte {
	return []byte(InscribePrivKeyString(privkey))
}

func ExtractPrivKeyString(tx []byte) string {
	return string(ExtractPrivKeyBytes(tx))
}

// Inscribe, extract pubkey

func ExtractPubKeyEd25519(tx []byte) (pubkey PubKeyEd25519) {
	bytes := re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
	copy(pubkey[:], bytes)
	return
}

func ExtractPubKeyBytes(tx []byte) []byte {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(tx)[1]
}

func InscribePubKeyString(pubkey PubKeyEd25519) string {
	return fmt.Sprintf("PubKeyEd25519{%X}", pubkey[:])
}

func InscribePubKeyBytes(pubkey PubKeyEd25519) []byte {
	return []byte(InscribePubKeyString(pubkey))
}

func ExtractPubKeyString(tx []byte) string {
	return string(ExtractPubKeyBytes(tx))
}
