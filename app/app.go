package app

import (
	"bytes"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	sm "github.com/zballs/comit/state"
	. "github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
	"strings"
	"time"
)

const version = "1.0.0"

type App struct {
	cli    *Client
	state  *sm.State
	cache  *sm.State
	Issues []string
}

func NewApp(cli *Client) *App {
	state := sm.NewState(cli)
	return &App{
		cli:   cli,
		state: state,
		cache: nil,
	}
}

// State filters

func (app *App) FilterFunc(filters []string) func(data []byte) bool {
	return app.state.FilterFunc(filters)
}

func (app *App) SetFilters(filters []string) {
	app.state.SetBloomFilters(filters)
}

// Search pipeline

// We iterate through indices and check if each key's in the filters
// If a key k is in the filters, than we can xor the form ID
// with issue and location, convert the bytes to a time and
// check whether it's in range...
// Then we get values for the keys selected, the values being
// forms themselves or IPFS content ids for submitted forms

// TODO: test

func (app *App) IterQueryIndex(fun func([]byte) bool, in chan []byte) {

	query := EmptyQuery(QuerySize)
	result := app.Query(query)

	if result.IsErr() {
		panic(result.Error())
	}
	var size int
	wire.ReadBinaryBytes(result.Data, &size)

	for i := 0; i < size; i++ {

		query := IndexQuery(i)

		result := app.Query(query)

		if result.IsErr() {
			// handle
			continue
		}

		if fun(result.Data) {
			in <- result.Data
		}
	}

	close(in)
}

func (app *App) IterQueryValue(fun func([]byte) bool, in, out chan []byte) {

	for {

		data, more := <-in

		if !more {
			break
		}

		query := KeyQuery(data, QueryValue)

		result := app.Query(query)

		if result.IsErr() {
			// handle
			continue
		}

		if fun(result.Data) {
			out <- data
		}
	}

	close(out)
}

// Check if time's in range

func TimeRangeFunc(afterTime, beforeTime time.Time) func([]byte) bool {

	return func(data []byte) bool {

		var form Form

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
		app.Issues = append(app.Issues, value)
		return "Success"
	case "account":
		var err error
		var acc *Account
		wire.ReadJSONPtr(&acc, []byte(value), &err)
		if err != nil {
			return "Error decoding acc message: " + err.Error()
		}
		app.state.SetAccount(acc.PubKey.Address(), acc)
		return "Success"
	}
	return "Unrecognized option key " + key
}

func (app *App) AppendTx(tx []byte) tmsp.Result {
	var action Action
	err := wire.ReadBinaryBytes(tx, &action)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate and exec action tx
	res := sm.ExecuteAction(app.state, action, false)
	if res.IsErr() {
		return res.PrependLog("Error in AppendTx")
	}
	return res
}

func (app *App) CheckTx(tx []byte) tmsp.Result {
	var action Action
	err := wire.ReadBinaryBytes(tx, &action)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate and exec action
	res := sm.ExecuteAction(app.state, action, true)
	if res.IsErr() {
		return res.PrependLog("Error in CheckTx")
	}
	return res
}

func (app *App) Query(query []byte) tmsp.Result {

	queryType := query[0]

	switch queryType {

	case QueryValue, QueryIndex, QuerySize, QueryProof: // merkle-cli
		return app.cli.QuerySync(query)

	case QueryIssues:

		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(app.Issues, buf, &n, &err)
		if err != nil {
			return tmsp.ErrEncodingError.AppendLog("Failed to encode issues data")
		}
		data := buf.Bytes()
		return tmsp.NewResultOK(data, "")

	case QuerySearch:
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

		// Checks if forms are in filters
		fun1 := app.FilterFunc([]string{s.Issue})

		// Checks if forms are in time range
		fun2 := TimeRangeFunc(s.AfterTime, s.BeforeTime)

		in := make(chan []byte)
		out := make(chan []byte)

		go app.IterQueryIndex(fun1, in)
		go app.IterQueryValue(fun2, in, out)

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
		return tmsp.ErrUnknownRequest.AppendLog("Unrecognized query type")
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
	app.cache = app.state.CacheWrap()
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
