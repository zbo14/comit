package app

import (
	"github.com/tendermint/go-wire"
	tmspcli "github.com/tendermint/tmsp/client"
	tmsp "github.com/tendermint/tmsp/types"
)

type Client struct {
	tmspcli.Client
}

func NewClient(addr, tmsp string) (*Client, error) {
	if addr == "local" {
		return NewLocalClient(), nil
	}
	tmspClient, err := tmspcli.NewClient(addr, tmsp, false)
	if err != nil {
		return nil, err
	}
	return &Client{tmspClient}, nil
}

func NewLocalClient() *Client {
	merk := NewMerkleApp()
	tmspClient := tmspcli.NewLocalClient(nil, merk)
	return &Client{tmspClient}
}

func (cli *Client) GetSync(key []byte) (res tmsp.Result) {
	query := make([]byte, 1+wire.ByteSliceSize(key))
	buf := query
	buf[0] = 0x01
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	res = cli.QuerySync(query)
	if res.IsErr() {
		return res
	}
	value := res.Data
	return tmsp.NewResultOK(value, "")
}

func (cli *Client) SetSync(key []byte, value []byte) (res tmsp.Result) {
	txBytes := make([]byte, wire.ByteSliceSize(key)+wire.ByteSliceSize(value))
	buf := txBytes
	n, err := wire.PutByteSlice(buf, key)
	if err != nil {
		return tmsp.ErrInternalError.SetLog(
			"Error encoding key byteslice: " + err.Error())
	}
	buf = buf[n:]
	n, err = wire.PutByteSlice(buf, value)
	if err != nil {
		return tmsp.ErrInternalError.SetLog(
			"Error encoding value byteslice: " + err.Error())
	}
	return cli.AppendTxSync(txBytes)
}

// RemSync

//==============================================================//

func (cli *Client) Get(key []byte) (value []byte) {
	res := cli.GetSync(key)
	if res.IsErr() {
		panic(res.Error())
	}
	return res.Data
}

func (cli *Client) Set(key []byte, value []byte) {
	res := cli.SetSync(key, value)
	if res.IsErr() {
		panic(res.Error())
	}
}
