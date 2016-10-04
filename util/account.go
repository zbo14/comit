package util

import (
	"github.com/tendermint/go-crypto"
	bcrypt "golang.org/x/crypto/bcrypt"
)

// Account keys

func GenerateSecret(passwordBytes []byte) []byte {
	secret, _ := bcrypt.GenerateFromPassword(passwordBytes, 0)
	return secret
}

func CreateKeys(passwordBytes []byte) (crypto.PubKeyEd25519, crypto.PrivKeyEd25519) {
	secret := GenerateSecret(passwordBytes)
	privKey := crypto.GenPrivKeyEd25519FromSecret(secret)
	pubKey := privKey.PubKey().(crypto.PubKeyEd25519)
	return pubKey, privKey
}
