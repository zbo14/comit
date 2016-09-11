package util

import (
	"fmt"
	re "regexp"
)

// Pubkeys

func ReadPubKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`pubkey{([a-z0-9]{64})}`).FindSubmatch(bytes)[1]
}

func WritePubKeyBytes(pubKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("pubkey{%x}", pubKeyBytes))
}

func RemovePubKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`pubkey{([a-z0-9]{64})}`).FindIndex(bytes)[0]]
}

// Privkeys

func ReadPrivKeyBytes(bytes []byte) []byte {
	return re.MustCompile(`privkey{([a-z0-9]{128})}`).FindSubmatch(bytes)[1]
}

func WritePrivKeyBytes(privKeyBytes []byte) []byte {
	return []byte(fmt.Sprintf("privkey{%x}", privKeyBytes))
}

func RemovePrivKeyBytes(bytes []byte) []byte {
	return bytes[:re.MustCompile(`privkey{([a-z0-9]{128})}`).FindIndex(bytes)[0]]
}
