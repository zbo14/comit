package util

import "fmt"

func BytesToHexstr(bytes []byte) string {
	return fmt.Sprintf("%X", bytes)
}
