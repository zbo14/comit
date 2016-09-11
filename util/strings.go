package util

import (
	"fmt"
	. "github.com/tendermint/go-crypto"
	re "regexp"
	"strings"
)

// Pubkeys

func PubKeyToString(pubkey PubKeyEd25519) string {
	return fmt.Sprintf("%x", pubkey[:])
}

func WritePubKeyString(pubKeyString string) string {
	return fmt.Sprintf("PubKeyEd25519{%v}", pubKeyString)
}

func ReadPubKeyString(str string) string {
	res := re.MustCompile(`PubKeyEd25519{(.*?)}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
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
	res := re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func RemovePrivKeyString(str string) string {
	return str[:re.MustCompile(`PrivKeyEd25519{.*?}`).FindStringIndex(str)[0]]
}

// Form IDs

func ReadFormID(str string) string {
	res := re.MustCompile(`form{(.*?)}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func WriteFormID(str string) string {
	return fmt.Sprintf("form{%v}", str)
}

// Substring Match

func SubstringMatch(substr string, str string) bool {
	match := re.MustCompile(strings.ToLower(substr)).FindString(strings.ToLower(str))
	if len(match) > 0 {
		return true
	}
	return false
}
