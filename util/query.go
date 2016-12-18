package util

import "github.com/tendermint/go-wire"

const (
	QueryChainID byte = 0

	QueryValue byte = 1
	QueryIndex byte = 2
	QuerySize  byte = 3
	QueryProof byte = 4

	// App specfic
	QueryIssues byte = 5
	QuerySearch byte = 6
)

func EmptyQuery(QueryType byte) []byte {
	return []byte{QueryType}
}

func KeyQuery(key []byte, queryType byte) []byte {
	query := make([]byte, wire.ByteSliceSize(key)+1)
	buf := query
	buf[0] = queryType
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	return query
}

func IndexQuery(i int) []byte {
	query := make([]byte, 100)
	buf := query
	buf[0] = QueryIndex
	buf = buf[1:]
	n, err := wire.PutVarint(buf, i)
	if err != nil {
		return nil
	}
	query = query[:n+1]
	return query
}
