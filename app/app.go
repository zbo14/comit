package app

import (
	"bytes"
	"errors"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	sm "github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	"strings"
)

const (
	version = "0.1"
	//maxTxSize = 10240

	QUERY_CHAIN_ID byte = 0
	QUERY_SIZE     byte = 1
	QUERY_BY_KEY   byte = 2
	QUERY_BY_INDEX byte = 3
	QUERY_ISSUES   byte = 4

	CHECK_TX_CONNECT  byte = 5
	CHECK_TX_REGISTER byte = 6
	CHECK_TX_INBOX    byte = 7
	CHECK_TX_UPDATE   byte = 8
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
	res := app.Query([]byte{QUERY_SIZE})
	var s int
	wire.ReadBinaryBytes(res.Data, &s)
	return s
}

func (app *App) FilterFunc(filters []string, includes []bool) func(data []byte) bool {
	return app.state.FilterFunc(filters, includes)
}

func (app *App) SetFilters(filters []string) {
	app.state.SetBloomFilters(filters)
}

/*
// Run in goroutine
func (app *App) Iterate(fun func(data []byte) bool, in chan []byte) { //errs chan error
	for i := 0; i < app.GetSize(); i++ {
		res := app.QueryByIndex(i)
		if res.IsErr() {
			fmt.Println(res.Error())
			// errs <- errors.New(res.Error())
		}
		if fun(res.Data) {
			in <- res.Data
		}
	}
	close(in)
	// close(errs)
}

func (app *App) IterateNext(fun func(data []byte) bool, in, out chan []byte) {
	for {
		data, more := <-in
		if more {
			if fun(data) {
				// fmt.Printf("%X\n", data)
				out <- data
			}
		} else {
			break
		}
	}
	close(out)
}
*/

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

func (app *App) IsAdmin(addr []byte) bool {
	acc := app.state.GetAccount(addr)
	return acc.IsAdmin()
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
		// fmt.Printf("%X\n", acc.PubKey.Address())
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

	// Decode Tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate and exec tx
	res := sm.ExecTx(app.state, tx, false)
	if res.IsErr() {
		return res.PrependLog("Error in AppendTx")
	}
	// If RemoveAccountTx, remove account
	if tx.Type == types.RemoveAccountTx || tx.Type == types.RemoveAdminTx {
		key := res.Data
		app.cli.Remove(key)
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

	switch tx.Data[0] {

	case CHECK_TX_CONNECT:

		return tmsp.OK

	case CHECK_TX_REGISTER:

		data := make([]byte, wire.ByteSliceSize(tx.Input.Address)+1)
		buf := data
		buf[0] = CHECK_TX_REGISTER
		buf = buf[1:]
		wire.PutByteSlice(buf, tx.Input.Address)

		return tmsp.NewResultOK(data, "")

	case CHECK_TX_INBOX:

		data := make([]byte, wire.ByteSliceSize(tx.Input.Address)+1)
		buf := data
		buf[0] = CHECK_TX_INBOX
		buf = buf[1:]
		wire.PutByteSlice(buf, tx.Input.Address)

		return tmsp.NewResultOK(data, "")

	default:
		return tmsp.ErrUnknownRequest

	}
}

func (app *App) Query(query []byte) tmsp.Result {
	switch query[0] {
	case QUERY_CHAIN_ID:
		return tmsp.NewResultOK(nil, app.GetChainID())
	case QUERY_ISSUES:
		buf, n, err := new(bytes.Buffer), int(0), error(nil)
		wire.WriteBinary(app.issues, buf, &n, &err)
		return tmsp.NewResultOK(buf.Bytes(), "")
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
