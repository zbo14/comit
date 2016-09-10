package util

import (
	"fmt"
	re "regexp"
)

// Pubkeys

func ReadPubKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`PubKeyEd25519{(.*?)}`).FindSubmatch(bytes)[1]
}

func WritePubKeyBytes(pubKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("PubKeyEd25519{%x}", pubKeyBytes))
}

func RemovePubKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`PubKeyEd25519{.*?}`).FindIndex(bytes)[0]]
}

// Privkeys

func ReadPrivKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`PrivKeyEd25519{(.*?)}`).FindSubmatch(bytes)[1]
}

func WritePrivKeyBytes(privKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("PrivKeyEd25519{%x}", privKeyBytes))
}

func RemovePrivKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`PrivKeyEd25519{.*?}`).FindIndex(bytes)[0]]
}
