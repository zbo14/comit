package util

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/tendermint/go-crypto"
	bcrypt "golang.org/x/crypto/bcrypt"
)

const PUBKEY_LENGTH = 32
const PRIVKEY_LENGTH = 64

// Generate secret from password string

func GenerateSecret(password string) ([]byte, error) {
	secret, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// Generate keypair from password string

func GenerateKeypair(password string) (crypto.PubKey, crypto.PrivKey, error) {
	secret, err := GenerateSecret(password)
	if err != nil {
		return nil, nil, err
	}
	privKey := crypto.GenPrivKeyEd25519FromSecret(secret)
	pubKey := privKey.PubKey().(crypto.PubKeyEd25519)
	return pubKey, privKey, nil
}

// Keys to hex strings

func PubKeytoHexstr(pubKey crypto.PubKey) string {
	pubKeyEd25519 := pubKey.(crypto.PubKeyEd25519)
	return BytesToHexstr(pubKeyEd25519[:])
}

func PrivKeytoHexstr(privKey crypto.PrivKey) string {
	privKeyEd25519 := privKey.(crypto.PrivKeyEd25519)
	return BytesToHexstr(privKeyEd25519[:])
}

// Hex strings to keys

func PubKeyfromHexstr(pubKeystr string) (crypto.PubKey, error) {
	var pubKey crypto.PubKeyEd25519
	pubKeyBytes, err := hex.DecodeString(pubKeystr)
	if err != nil || len(pubKeyBytes) != PUBKEY_LENGTH {
		return nil, errors.New("Invalid public key")
	}
	copy(pubKey[:], pubKeyBytes)
	return pubKey, nil
}

func PrivKeyfromHexstr(privKeystr string) (crypto.PrivKey, error) {
	var privKey crypto.PrivKeyEd25519
	privKeyBytes, err := hex.DecodeString(privKeystr)
	if err != nil || len(privKeyBytes) != PRIVKEY_LENGTH {
		return nil, errors.New("Invalid private key")
	}
	copy(privKey[:], privKeyBytes)
	return privKey, nil
}
