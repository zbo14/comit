package util

import (
	"encoding/hex"
	"fmt"
)

func BytesToHexstr(bytes []byte) string {
	return fmt.Sprintf("%X", bytes)
}

func HexstrToBytes(hexstr string) []byte {
	bytes, _ := hex.DecodeString(hexstr)
	return bytes
}
