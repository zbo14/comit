package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	ntwk "github.com/zballs/3ii/network"
	sm "github.com/zballs/3ii/state"
	"github.com/zballs/3ii/types"
	"strings"
)

const (
	version   = "0.1"
	maxTxSize = 10240

	querySize    byte = 1
	queryByKey   byte = 2
	queryByIndex byte = 3

	deptChID  byte = 10
	adminChID byte = 20
)

type App struct {
	cli        *Client
	state      *sm.State
	cacheState *sm.State

	depts     *ntwk.MyReactor
	deptChIDs map[string]byte
	deptChID  byte

	admins     *ntwk.MyReactor
	adminChIDs map[string]byte
	adminChID  byte
}

func NewApp(cli *Client) *App {
	state := sm.NewState(cli)
	return &App{
		cli:        cli,
		state:      state,
		cacheState: nil,

		deptChIDs: make(map[string]byte),
		deptChID:  deptChID,

		adminChIDs: make(map[string]byte),
		adminChID:  adminChID,
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
	res := app.Query([]byte{querySize})
	var s int
	wire.ReadBinaryBytes(res.Data, &s)
	return s
}

func (app *App) FilterFunc(filters []string) func(data []byte) bool {
	return app.state.FilterFunc(filters)
}

func (app *App) QueryByKey(key []byte) tmsp.Result {
	query := make([]byte, wire.ByteSliceSize(key)+1)
	buf := query
	buf[0] = queryByKey
	buf = buf[1:]
	wire.PutByteSlice(buf, key)
	res := app.Query(query)
	return res
}

func (app *App) QueryByIndex(i int) tmsp.Result {
	query := make([]byte, 100)
	buf := query
	buf[0] = queryByIndex
	buf = buf[1:]
	n, err := wire.PutVarint(buf, i)
	if err != nil {
		return tmsp.ErrInternalError
	}
	query = query[:n+1]
	res := app.Query(query)
	return res
}

// Run in goroutine
func (app *App) Iterate(fun func(data []byte) bool, in chan []byte, errs chan error) {
	for i := 0; i < app.GetSize(); i++ {
		res := app.QueryByIndex(i)
		if res.IsErr() {
			errs <- errors.New(res.Error())
		}
		if !fun(res.Data) {
			continue
		}
		in <- res.Data
	}
	close(in)
	close(errs)
}

func (app *App) IterateNext(fun func(data []byte) bool, in, out chan []byte) {
	for {
		data, more := <-in
		if more {
			if !fun(data) {
				continue
			}
			out <- data
		} else {
			break
		}
	}
	close(out)
}

func (app *App) AddDept(dept string) {
	app.deptChIDs[dept] = app.deptChID
	app.deptChID++
}

func (app *App) DeptChIDs() map[string]byte {
	return app.deptChIDs
}

func (app *App) DeptChID(dept string) byte {
	return app.deptChIDs[dept]
}

func (app *App) CreateDeptReactor() *ntwk.MyReactor {
	chDescs := ntwk.CreateChDescs(app.deptChIDs)
	app.depts = ntwk.NewReactor(chDescs, true)
	return app.depts
}

func (app *App) AddAdmin(pubKeyString string) {
	app.adminChIDs[pubKeyString] = app.adminChID
	app.adminChID++
}

func (app *App) AdminChIDs() map[string]byte {
	return app.adminChIDs
}

func (app *App) AdminChID(pubKeyString string) byte {
	return app.adminChIDs[pubKeyString]
}

func (app *App) CreateAdminReactor() *ntwk.MyReactor {
	chDescs := ntwk.CreateChDescs(app.adminChIDs)
	app.admins = ntwk.NewReactor(chDescs, true)
	return app.admins
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
	case "dept":
		app.AddDept(value)
		return "Success"
	case "admin":
		var err error
		var acc *types.Account
		wire.ReadJSONPtr(&acc, []byte(value), &err)
		if err != nil {
			return "Error decoding acc message: " + err.Error()
		}
		fmt.Printf("%X\n", acc.PubKey.Address())
		app.state.SetAccount(acc.PubKey.Address(), acc)
		app.AddAdmin(acc.PubKey.KeyString())
		return "Success"
	}
	return "Unrecognized option key " + key
}

func (app *App) AppendTx(txBytes []byte) tmsp.Result {
	if len(txBytes) > maxTxSize {
		return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}
	// Decode Tx
	var tx = types.Tx{}
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
	return app.cli.QuerySync(query)
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
