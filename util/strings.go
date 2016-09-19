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
	return fmt.Sprintf("pubkey {%v}", pubKeyString)
}

func ReadPubKeyString(str string) string {
	res := re.MustCompile(`pubkey {([a-z0-9]{64})}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func RemovePubKeyString(str string) string {
	return str[:re.MustCompile(`pubkey {([a-z0-9]{64})}`).FindStringIndex(str)[0]]
}

// Privkeys

func PrivKeyToString(privkey PrivKeyEd25519) string {
	return fmt.Sprintf("%x", privkey[:])
}

func WritePrivKeyString(privKeyString string) string {
	return fmt.Sprintf("privkey {%v}", privKeyString)
}

func ReadPrivKeyString(str string) string {
	res := re.MustCompile(`privkey {([a-z0-9]{128})}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func RemovePrivKeyString(str string) string {
	return str[:re.MustCompile(`privkey {([a-z0-9]{128})}`).FindStringIndex(str)[0]]
}

// Passphrase

func WritePassphrase(passphrase string) string {
	return fmt.Sprintf("passphrase {%v}", passphrase)
}

func ReadPassphrase(str string) string {
	res := re.MustCompile(`passphrase {(.*?)}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func RemovePassphrase(str string) string {
	return str[:re.MustCompile(`passphrase {(.*?)}`).FindStringIndex(str)[0]]
}

// Form IDs

func ReadFormID(str string) string {
	res := re.MustCompile(`form {([a-z0-9]{32})}`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func WriteFormID(str string) string {
	return fmt.Sprintf("form {%v}", str)
}

// Substring Match

func SubstringMatch(substr string, str string) bool {
	match := re.MustCompile(strings.ToLower(substr)).FindString(strings.ToLower(str))
	if len(match) > 0 {
		return true
	}
	return false
}

// Regex Formatting

func RegexQuestionMarks(str string) string {
	return `` + strings.Replace(str, `?`, `\?`, -1)
}
