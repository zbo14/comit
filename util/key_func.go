package util

import (
	"fmt"
	. "github.com/tendermint/go-crypto"
	re "regexp"
)

// Pubkey funcs

func PubKeyToString(pubkey PubKeyEd25519) string {
	return fmt.Sprintf("%x", pubkey[:])
}

func ReadPubKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(bytes)[1]
}

func WritePubKeyBytes(pubKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("PubKeyEd25519{%x}", pubKeyBytes))
}

func WritePubKeyString(pubKeyString string) string {
	return fmt.Sprintf("PubKeyEd25519{%v}", pubKeyString)
}

func ReadPubKeyString(str string) string {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindStringSubmatch(str)[1]
}

func RemovePubKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`PubKeyEd25519{.*?}`).FindIndex(bytes)[0]]
}

func RemovePubKeyString(str string) string {
	return str[:re.MustCompile(`PubKeyEd25519{.*?}`).FindStringIndex(str)[0]]
}

// Privkey funcs

func PrivKeyToString(privkey PrivKeyEd25519) string {
	return fmt.Sprintf("%x", privkey[:])
}

func ReadPrivKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(bytes)[1]
}

func WritePrivKeyBytes(privKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("PrivKeyEd25519{%x}", privKeyBytes))
}

func WritePrivKeyString(privKeyString string) string {
	return fmt.Sprintf("PrivKeyEd25519{%v}", privKeyString)
}

func ReadPrivKeyString(str string) string {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindStringSubmatch(str)[1]
}

func RemovePrivKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`PrivKeyEd25519{.*?}`).FindIndex(bytes)[0]]
}

func RemovePrivKeyString(str string) string {
	return str[:re.MustCompile(`PrivKeyEd25519{.*?}`).FindStringIndex(str)[0]]
}
