package app

import (
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	sm "github.com/zballs/3ii/state"
	"github.com/zballs/3ii/types"
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

// TMSP requests

func (app *App) Info() string {
	return fmt.Sprintf("3ii v%v", version)
}

func (app *App) SetOption(key string, value string) (log string) {
	return "No options are supported yet"
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
	return tmsp.OK
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
	res := sm.ExecTx(app.cacheState, tx, true)
	if res.IsErr() {
		return res.PrependLog("Error in CheckTx")
	}
	return tmsp.OK
}

func (app *App) Query(query []byte) tmsp.Result {
	return app.cli.QuerySync(query)
}

func (app *App) Commit() (res tmsp.Result) {
	res = app.cli.CommitSync()
	if res.IsErr() {
		PanicSanity("Error getting hash: " + res.Error())
	}
	return res
}
