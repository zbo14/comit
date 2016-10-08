package app

import (
	"errors"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	sm "github.com/zballs/3ii/state"
	"github.com/zballs/3ii/types"
	"strings"
)

const (
	version   = "0.1"
	maxTxSize = 10240
)

type App struct {
	cli        *Client
	state      *sm.State
	cacheState *sm.State
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

// TMSP requests

func (app *App) Info() string {
	return Fmt("3ii v%v", version)
}

func (app *App) SetOption(key string, value string) (log string) {
	_, key = splitKey(key)
	switch key {
	case "chainID":
		app.state.SetChainID(value)
		return "Success"
	case "account":
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
	if len(txBytes) > maxTxSize {
		return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}
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
	if tx.Type == types.RemoveAccountTx {
		key := res.Data
		app.cli.Remove(key)
	}
	return res
}

func (app *App) CheckTx(txBytes []byte) tmsp.Result {
	if len(txBytes) > maxTxSize {
		return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}
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
	return tmsp.OK
}

func (app *App) Query(query []byte) tmsp.Result {
	if len(query) == 0 {
		return tmsp.ErrEncodingError.SetLog("Query cannot be zero length")
	}
	typeByte := query[0]
	query = query[1:]
	switch typeByte {
	case 0x01: // Size
		return app.cli.SizeSync()
	case 0x02: // Query by key
		return app.cli.GetSync(query)
	default:
		return tmsp.ErrUnknownRequest.SetLog(
			Fmt("Unexpected Query type byte %X", typeByte))
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
