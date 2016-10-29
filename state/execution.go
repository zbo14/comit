package state

import (
	"bytes"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/lib"
	"github.com/zballs/comit/types"
	. "github.com/zballs/comit/util"
)

const (
	ConnectAccount byte = 0
	ConnectAdmin   byte = 1
)

// If the tx is invalid, a TMSP error will be returned.
func ExecTx(state *State, tx types.Tx, isCheckTx bool) (res tmsp.Result) {

	chainID := state.GetChainID()

	// Validate Input Basic
	res = tx.Input.ValidateBasic()
	if res.IsErr() {
		return res
	}

	var inAcc *types.Account

	if tx.Type == types.CreateAccountTx {
		// Create new account
		// Must have txIn pubKey
		inAcc = types.NewAccount(tx.Input.PubKey, 0)
	} else {
		// Get input account
		inAcc = state.GetAccount(tx.Input.Address)
		if inAcc == nil {
			return tmsp.ErrBaseUnknownAddress
		}
		if tx.Input.PubKey != nil {
			inAcc.PubKey = tx.Input.PubKey
		}
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(inAcc, signBytes, tx.Input)
	if res.IsErr() {
		log.Info(Fmt("validateInputAdvanced failed on %X: %v", tx.Input.Address, res))
		return res.PrependLog("in validateInputAdvanced()")
	}

	inAcc.Sequence += 1

	// If CheckTx, we are done.
	if isCheckTx {
		// Well, not quite
		if tx.Data[0] == ConnectAdmin && !inAcc.IsAdmin() {
			return tmsp.ErrUnauthorized
		}
		state.SetAccount(tx.Input.Address, inAcc)
		return tmsp.OK
	}

	// Create inAcc checkpoint
	inAccCopy := inAcc.Copy()

	// Run the tx.
	cacheState := state.CacheWrap()
	cacheState.SetAccount(tx.Input.Address, inAcc)
	ctx := types.NewCallContext(tx.Input.Address)
	switch tx.Type {
	case types.CreateAccountTx:
		res = RunCreateAccountTx(cacheState, ctx, tx.Data)
	case types.RemoveAccountTx:
		res = RunRemoveAccountTx(cacheState, ctx, tx.Data)
	case types.SubmitTx:
		res = RunSubmitTx(cacheState, ctx, tx.Data)
	case types.ResolveTx:
		if !inAcc.PermissionToResolve() {
			fmt.Println("Not granted permission to resolve")
			res = tmsp.ErrUnauthorized
		} else {
			res = RunResolveTx(cacheState, ctx, tx.Data)
		}
	case types.CreateAdminTx:
		if !inAcc.PermissionToCreateAdmin() {
			res = tmsp.ErrUnauthorized
		} else {
			res = RunCreateAdminTx(cacheState, ctx, tx.Data)
		}
	default:
		res = tmsp.ErrUnknownRequest.SetLog(
			Fmt("Error unrecognized tx type: %v", tx.Type))
	}
	if res.IsOK() {
		cacheState.CacheSync()
		log.Info("Successful execution")
	} else {
		log.Info("AppTx failed", "error", res)
		cacheState.SetAccount(tx.Input.Address, inAccCopy)
	}
	return res
}

//=====================================================================//

func RunCreateAccountTx(state *State, ctx types.CallContext, data []byte) tmsp.Result {
	// Just return OK
	return tmsp.OK
}

func RunCreateAdminTx(state *State, ctx types.CallContext, data []byte) tmsp.Result {

	// Create keys
	pubKey, privKey := CreateKeys(data)

	// Create new admin
	newAcc := types.NewAdmin(pubKey)
	state.SetAccount(pubKey.Address(), newAcc)

	// Return privKey
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(&privKey, buf, &n, &err)
	return tmsp.NewResultOK(buf.Bytes(), "")
}

func RunRemoveAccountTx(state *State, ctx types.CallContext, data []byte) tmsp.Result {
	// Return key so we can remove in AppendTx
	key := AccountKey(ctx.Caller)
	return tmsp.NewResultOK(key, "")
}

func RunSubmitTx(state *State, ctx types.CallContext, data []byte) (res tmsp.Result) {
	var form lib.Form
	err := wire.ReadBinaryBytes(data, &form)
	if err != nil {
		return tmsp.NewResult(
			lib.ErrDecodeForm, nil, Fmt("Error: could not decode form data: %v", data))
	}
	issue := form.Issue
	formID := (&form).ID()
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(formID, buf, &n, &err)
	state.Set(buf.Bytes(), data)
	err = state.AddToFilter(buf.Bytes(), issue)
	if err != nil {
		// False positive
		// print for now
		fmt.Println(err.Error())
	}
	err = state.AddToFilter(buf.Bytes(), "unresolved")
	if err != nil {
		// False positive
		// print for now
		fmt.Println(err.Error())
	}
	fmt.Printf("Added form to %s filter\n", issue)
	return tmsp.NewResultOK(formID, "")
}

func RunResolveTx(state *State, ctx types.CallContext, data []byte) (res tmsp.Result) {
	formID := BytesToHexString(data)
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(data, buf, &n, &err)
	value := state.Get(buf.Bytes())
	if len(value) == 0 {
		return tmsp.NewResult(
			lib.ErrFindForm, nil, Fmt("Error cannot find form with ID: %v", formID))
	}
	var form *lib.Form
	err = wire.ReadBinaryBytes(value, form)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error parsing form bytes: %v", err.Error()))
	}
	timestr := TimeString()
	pubKeyString := BytesToHexString(ctx.Caller)
	err = form.Resolve(timestr, pubKeyString)
	if err != nil {
		return tmsp.NewResult(
			lib.ErrFormAlreadyResolved, nil, Fmt("Error already resolved form with ID: %v", formID))
	}
	buf, n, err = new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(form, buf, &n, &err)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error encoding form with ID: %v", formID))
	}
	state.Set(data, buf.Bytes())
	err = state.AddToFilter(data, "resolved")
	if err != nil {
		// OK, false positive
		// print for now
		fmt.Println(err.Error())
	}
	return tmsp.OK
}

//===============================================================================================//

func validateInputAdvanced(acc *types.Account, signBytes []byte, in types.TxInput) (res tmsp.Result) {
	// Check sequence
	seq := acc.Sequence
	if seq+1 != in.Sequence {
		return tmsp.ErrBaseInvalidSequence.AppendLog(
			Fmt("Got %v, expected %v. (acc.seq=%v)", in.Sequence, seq+1, acc.Sequence))
	}
	// Check signatures
	if !acc.PubKey.VerifyBytes(signBytes, in.Signature) {
		return tmsp.ErrBaseInvalidSignature.AppendLog(
			Fmt("SignBytes: %X", signBytes))
	}
	return tmsp.OK
}
