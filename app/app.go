package app

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/forms"
	sm "github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"strings"
	"time"
)

const (
	version = "0.1"
	//maxTxSize = 10240

	QueryChainID byte = 0
	QuerySize    byte = 1
	QueryKey     byte = 2
	QueryIndex   byte = 3
	QueryIssues  byte = 4
	QuerySearch  byte = 5
)

type App struct {
	cli        *Client
	state      *sm.State
	cacheState *sm.State
	issues     []string
}

func NewApp(cli *Client) *App {
	state := sm.NewState(cli)
	return &App{
		cli:        cli,
		state:      state,
		cacheState: nil,
	}
}

// Get

func (app *App) GetChainID() string {
	return app.state.GetChainID()
}

func (app *App) GetSequence(addr []byte) (int, error) {
	acc := app.state.GetAccount(addr)
	if acc == nil {
		return 0, errors.New("Error could not find account")
	}
	return acc.Sequence, nil
}

func (app *App) GetSize() int {
	res := app.Query([]byte{QuerySize})
	s := binary.BigEndian.Uint32(res.Data)
	return int(s)
}

// Filter

func (app *App) FilterFunc(filters []string) func(data []byte) bool {
	return app.state.FilterFunc(filters)
}

func (app *App) SetFilters(filters []string) {
	app.state.SetBloomFilters(filters)
}

// Query

func (app *App) QueryByKey(key []byte) tmsp.Result {
	query := make([]byte, wire.ByteSliceSize(key)+1)
	buf := query
	buf[0] = QueryKey
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	return app.Query(query)
}

func (app *App) QueryByIndex(i int) tmsp.Result {
	query := make([]byte, 100)
	buf := query
	buf[0] = QueryIndex
	buf = buf[1:]
	n, err := wire.PutVarint(buf, i)
	if err != nil {
		return tmsp.ErrEncodingError
	}
	query = query[:n+1]
	return app.Query(query)
}

func (app *App) IterQueryIndex(fun func([]byte) bool, in chan []byte) {

	for i := 0; i < app.GetSize(); i++ {

		result := app.QueryByIndex(i)

		if result.IsErr() {
			panic(result.Error())
		}

		if fun(result.Data) {
			fmt.Printf("%X\n", result.Data)
			in <- result.Data
		}
	}

	close(in)
}

func (app *App) IterQueryKey(fun func([]byte) bool, in, out chan []byte) {

	for {

		data, more := <-in

		if !more {
			break
		}

		result := app.QueryByKey(data)

		if result.IsErr() {
			panic(result.Error())
		}

		if fun(result.Data) {
			fmt.Printf("%X\n", data)
			out <- data
		}
	}

	close(out)
}

// Issues

func (app *App) Issues() []string {
	return app.issues
}

func (app *App) IsIssue(issue string) bool {
	for _, i := range app.issues {
		if issue == i {
			return true
		}
	}
	return false
}

// Admin

func (app *App) IsAdmin(addr []byte) bool {
	acc := app.state.GetAccount(addr)
	return acc.IsAdmin()
}

// Check if time's in range

func TimeRangeFunc(afterTime, beforeTime time.Time) func([]byte) bool {

	return func(data []byte) bool {

		var form forms.Form

		err := wire.ReadBinaryBytes(data, &form)

		if err != nil {
			panic(err)
		}

		t := ParseMinuteString(form.SubmittedAt)

		if t.After(afterTime) && t.Before(beforeTime) {
			return true
		}

		return false
	}
}

// TMSP requests

func (app *App) Info() string {
	return Fmt("comit v%v", version)
}

func (app *App) SetOption(key string, value string) (log string) {
	_, key = splitKey(key)
	switch key {
	case "chainID":
		app.state.SetChainID(value)
		return "Success"
	case "issue":
		app.issues = append(app.issues, value)
		return "Success"
	case "admin":
		var err error
		var acc *types.Account
		wire.ReadJSONPtr(&acc, []byte(value), &err)
		if err != nil {
			return "Error decoding acc message: " + err.Error()
		}
		app.state.SetAccount(acc.PubKey.Address(), acc)
		return "Success"
	}
	return "Unrecognized option key " + key
}

func (app *App) AppendTx(txBytes []byte) tmsp.Result {

	/*
		if len(txBytes) > maxTxSize {
			return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
		}
	*/

	// Create tx
	var tx types.Tx

	// Decode Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)

	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	// Validate and exec tx
	res := sm.ExecTx(app.state, tx, false)
	if res.IsErr() {
		return res.PrependLog("Error in AppendTx")
	}

	switch tx.Type {

	case types.RemoveAccountTx:
		key := res.Data
		app.cli.Remove(key)

	case types.RemoveAdminTx:
		key := res.Data
		app.cli.Remove(key)

	default:
	}

	return res
}

func (app *App) CheckTx(txBytes []byte) tmsp.Result {

	/*
		if len(txBytes) > maxTxSize {
			return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
		}
	*/

	// Decode tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate and exec tx
	res := sm.ExecTx(app.state, tx, true)
	if res.IsErr() {
		return res.PrependLog("Error in CheckTx")
	}

	switch tx.Type {

	case types.ConnectTx:

		return tmsp.OK

	case types.UpdateTx:

		data := make([]byte, wire.ByteSliceSize(tx.Input.Address)+1)
		buf := data
		buf[0] = types.UpdateTx
		buf = buf[1:]
		wire.PutByteSlice(buf, tx.Input.Address)

		return tmsp.NewResultOK(data, "")

	default:
		return tmsp.ErrUnknownRequest

	}
}

func (app *App) Query(query []byte) tmsp.Result {

	switch query[0] {

	case QueryChainID:

		return tmsp.NewResultOK(nil, app.GetChainID())

	case QueryIssues:

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(app.issues, buf, &n, &err)
		return tmsp.NewResultOK(buf.Bytes(), "")

	case QuerySearch:

		fmt.Println("search query")

		search, _, err := wire.GetByteSlice(query[1:])

		if err != nil {
			panic(err)
		}

		s := struct {
			AfterTime  time.Time
			BeforeTime time.Time
			Issue      string
		}{}

		err = wire.ReadBinaryBytes(search, &s)

		if err != nil {
			return tmsp.ErrEncodingError
		}

		fmt.Printf("%v\n", s)

		// Checks if forms are in filters..
		// for now, just check if forms have issue
		fun1 := app.FilterFunc([]string{s.Issue})

		// Checks if forms are in time range
		fun2 := TimeRangeFunc(s.AfterTime, s.BeforeTime)

		in := make(chan []byte)
		out := make(chan []byte)

		go app.IterQueryIndex(fun1, in)
		go app.IterQueryKey(fun2, in, out)

		var datas [][]byte

		for {

			data, ok := <-out

			if !ok {
				break
			}

			datas = append(datas, data)
		}

		if len(datas) == 0 {
			return tmsp.NewResultOK(nil, "")
		}

		data := wire.BinaryBytes(datas)

		return tmsp.NewResultOK(data, "")

	default:
		return app.cli.QuerySync(query)
	}
}

func (app *App) Commit() (res tmsp.Result) {
	res = app.cli.CommitSync()
	if res.IsErr() {
		PanicSanity("Error getting hash: " + res.Error())
	}
	return res
}

// TMSP::InitChain
func (app *App) InitChain(validators []*tmsp.Validator) {
	app.cli.InitChainSync(validators)
}

// TMSP::BeginBlock
func (app *App) BeginBlock(height uint64) {
	app.cli.BeginBlockSync(height)
	app.cacheState = app.state.CacheWrap()
}

// TMSP::EndBlock
func (app *App) EndBlock(height uint64) (vz []*tmsp.Validator) {
	vz, _ = app.cli.EndBlockSync(height)
	return vz
}

// -----------------------------------------

func splitKey(key string) (prefix string, suffix string) {
	if strings.Contains(key, "/") {
		keyParts := strings.SplitN(key, "/", 2)
		return keyParts[0], keyParts[1]
	}
	return key, ""
}
