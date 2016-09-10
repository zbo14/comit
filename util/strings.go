package util

import (
	"fmt"
	. "github.com/tendermint/go-crypto"
	re "regexp"
)

// Pubkeys

func PubKeyToString(pubkey PubKeyEd25519) string {
	return fmt.Sprintf("%x", pubkey[:])
}

func WritePubKeyString(pubKeyString string) string {
	return fmt.Sprintf("PubKeyEd25519{%v}", pubKeyString)
}

func ReadPubKeyString(str string) string {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindStringSubmatch(str)[1]
}

func RemovePubKeyString(str string) string {
	return str[:re.MustCompile(`PubKeyEd25519{.*?}`).FindStringIndex(str)[0]]
}

// Privkeys

func PrivKeyToString(privkey PrivKeyEd25519) string {
	return fmt.Sprintf("%x", privkey[:])
}

func WritePrivKeyString(privKeyString string) string {
	return fmt.Sprintf("PrivKeyEd25519{%v}", privKeyString)
}

func ReadPrivKeyString(str string) string {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindStringSubmatch(str)[1]
}

func RemovePrivKeyString(str string) string {
	return str[:re.MustCompile(`PrivKeyEd25519{.*?}`).FindStringIndex(str)[0]]
}

// Form IDs

func ReadFormID(str string) string {
	return re.MustCompile(`form{(.*?)}`).FindStringSubmatch(str)[1]
}

func WriteFormID(str string) string {
	return fmt.Sprintf("form{%v}", str)
}
