package util

import (
	"encoding/hex"
	"fmt"
)

// Hex string (for map indexing)

func BytesToHexString(bytes []byte) string {
	return fmt.Sprintf("%X", bytes)
}

func HexStringToBytes(hexstr string) ([]byte, error) {
	bytes, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
