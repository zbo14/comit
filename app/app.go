package app

import (
	"errors"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	ntwk "github.com/zballs/comit/network"
	sm "github.com/zballs/comit/state"
	"github.com/zballs/comit/types"
	"strings"
)

const (
	version   = "0.1"
	maxTxSize = 10240

	querySize    byte = 1
	queryByKey   byte = 2
	queryByIndex byte = 3

	issueID byte = 10
	adminID byte = 20
)

type App struct {
	cli        *Client
	state      *sm.State
	cacheState *sm.State

	issues   *ntwk.MyReactor
	issueIDs map[string]byte
	issueID  byte

	admins   *ntwk.MyReactor
	adminIDs map[string]byte
	adminID  byte
}

func NewApp(cli *Client) *App {
	state := sm.NewState(cli)
	return &App{
		cli:        cli,
		state:      state,
		cacheState: nil,

		issueIDs: make(map[string]byte),
		issueID:  issueID,

		adminIDs: make(map[string]byte),
		adminID:  adminID,
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

func (app *App) SetFilters(filters []string) {
	app.state.SetBloomFilters(filters)
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
func (app *App) Iterate(fun func(data []byte) bool, in chan []byte) { //errs chan error
	for i := 0; i < app.GetSize(); i++ {
		res := app.QueryByIndex(i)
		if res.IsErr() {
			fmt.Println(res.Error())
			// errs <- errors.New(res.Error())
		}
		// data := make([]byte, len(res.Data))
		// copy(data, res.Data)
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
				fmt.Printf("%X\n", data)
				out <- data
			}
		} else {
			break
		}
	}
	close(out)
}

func (app *App) Issues() []string {
	issues := make([]string, len(app.issueIDs))
	idx := 0
	for issue, _ := range app.issueIDs {
		issues[idx] = issue
		idx++
	}
	return issues
}

func (app *App) AddIssue(issue string) {
	app.issueIDs[issue] = app.issueID
	app.issueID++
}

func (app *App) IssueIDs() map[string]byte {
	return app.issueIDs
}

func (app *App) IssueID(issue string) byte {
	return app.issueIDs[issue]
}

func (app *App) IsIssue(issue string) bool {
	return app.issueIDs[issue] > 0x0
}

func (app *App) CreateIssueReactor() *ntwk.MyReactor {
	chDescs := ntwk.CreateChDescs(app.issueIDs)
	app.issues = ntwk.NewReactor(chDescs, true)
	return app.issues
}

func (app *App) AddAdmin(pubKeyString string) {
	app.adminIDs[pubKeyString] = app.adminID
	app.adminID++
}

func (app *App) AdminIDs() map[string]byte {
	return app.adminIDs
}

func (app *App) AdminID(pubKeyString string) byte {
	return app.adminIDs[pubKeyString]
}

func (app *App) CreateAdminReactor() *ntwk.MyReactor {
	chDescs := ntwk.CreateChDescs(app.adminIDs)
	app.admins = ntwk.NewReactor(chDescs, true)
	return app.admins
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
	case "issue": //--> issues
		app.AddIssue(value)
		return "Success"
	case "admin": //--> admins
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
