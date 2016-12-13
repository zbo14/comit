package app

import (
	"github.com/tendermint/go-wire"
	tmspcli "github.com/tendermint/tmsp/client"
	tmsp "github.com/tendermint/tmsp/types"
	. "github.com/zballs/comit/util"
	"log"
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
	query := make([]byte, wire.ByteSliceSize(key)+1)
	buf := query
	buf[0] = QueryKey
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	res = cli.QuerySync(query)
	if res.IsErr() {
		return res
	}
	value := res.Data
	return tmsp.NewResultOK(value, "")
}

func (cli *Client) SetSync(key []byte, value []byte) tmsp.Result {
	txBytes := make([]byte, wire.ByteSliceSize(key)+wire.ByteSliceSize(value)+1)
	buf := txBytes
	buf[0] = 0x01
	buf = buf[1:]
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

func (cli *Client) RemSync(key []byte) tmsp.Result {
	tx := make([]byte, wire.ByteSliceSize(key)+1)
	buf := tx
	buf[0] = 0x02
	buf = buf[1:]
	_, err := wire.PutByteSlice(buf, key)
	if err != nil {
		return tmsp.ErrInternalError.SetLog(
			"encoding key byteslice: " + err.Error())
	}
	return cli.AppendTxSync(tx)
}

// ------------------------------------------------ //

func (client *Client) Get(key []byte) (value []byte) {
	res := client.GetSync(key)
	if res.IsErr() {
		log.Println(res.Error())
	}
	return res.Data
}

func (client *Client) Set(key []byte, value []byte) {
	res := client.SetSync(key, value)
	if res.IsErr() {
		log.Println(res.Error())
	}
}

func (client *Client) Remove(key []byte) {
	res := client.RemSync(key)
	if res.IsErr() {
		log.Println(res.Error())
	}
}
