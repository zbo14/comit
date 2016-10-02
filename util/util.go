package util

import (
	"encoding/hex"
	"fmt"
	"github.com/tendermint/go-crypto"
	bcrypt "golang.org/x/crypto/bcrypt"
	re "regexp"
	"strconv"
	"strings"
	"time"
)

// Hex string (for map indexing)

func BytesToHexString(bytes []byte) string {
	return fmt.Sprintf("%x", bytes)
}

func HexStringToBytes(hexstr string) ([]byte, error) {
	bytes, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Account keys

func GenerateSecret(passwordBytes []byte) []byte {
	secret, _ := bcrypt.GenerateFromPassword(passwordBytes, 0)
	return secret
}

func CreateKeys(passwordBytes []byte) (crypto.PubKeyEd25519, crypto.PrivKeyEd25519) {
	secret := GenerateSecret(passwordBytes)
	privKey := crypto.GenPrivKeyEd25519FromSecret(secret)
	pubKey := privKey.PubKey().(crypto.PubKeyEd25519)
	return pubKey, privKey
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

// Time

func TimeString() string {
	return time.Now().UTC().String()
}

// From js moment
func ParseTimeString(timestr string) time.Time {
	yr, _ := strconv.Atoi(timestr[:4])
	mo, _ := strconv.Atoi(timestr[5:7])
	d, _ := strconv.Atoi(timestr[8:10])
	hr, _ := strconv.Atoi(timestr[11:13])
	min, _ := strconv.Atoi(timestr[14:16])
	sec, _ := strconv.Atoi(timestr[17:19])
	return time.Date(yr, time.Month(mo), d, hr, min, sec, 0, time.UTC)
}

func DurationTimeStrings(timestr1, timestr2 string) time.Duration {
	tm1 := ParseTimeString(timestr1)
	tm2 := ParseTimeString(timestr2)
	return tm2.Sub(tm1)
}

func DurationHours(timestr1, timestr2 string) float64 {
	return DurationTimeStrings(timestr1, timestr2).Hours()
}

func DurationDays(timestr1, timestr2 string) float64 {
	return DurationHours(timestr1, timestr2) / float64(24)
}

func ToTheDay(timestr string) string {
	return timestr[:10]
}

func ToTheHour(timestr string) string {
	return timestr[:13]
}

func ToTheMinute(timestr string) string {
	return timestr[:16]
}

func ToTheSecond(timestr string) string {
	return timestr[:19]
}

// HTML

func ExtractText(str string) string {
	return re.MustCompile(`>(.*?)<`).FindStringSubmatch(str)[1]
}
