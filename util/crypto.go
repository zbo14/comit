package util

import (
	bcrypt "golang.org/x/crypto/bcrypt"
)

func GenerateSecret(passphrase []byte) []byte {
	secret, _ := bcrypt.GenerateFromPassword(passphrase, 0)
	return secret
}
