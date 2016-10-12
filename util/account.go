package util

import (
	"github.com/tendermint/go-crypto"
	bcrypt "golang.org/x/crypto/bcrypt"
)

// Account keys

func GenerateSecret(secretBytes []byte) []byte {
	secret, _ := bcrypt.GenerateFromPassword(secretBytes, 0)
	return secret
}

func CreateKeys(secretBytes []byte) (crypto.PubKeyEd25519, crypto.PrivKeyEd25519) {
	secret := GenerateSecret(secretBytes)
	privKey := crypto.GenPrivKeyEd25519FromSecret(secret)
	pubKey := privKey.PubKey().(crypto.PubKeyEd25519)
	return pubKey, privKey
}
