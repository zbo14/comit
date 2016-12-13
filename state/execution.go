package state

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	. "github.com/zballs/comit/types"
)

// Logger
var log = NewLogger("execution")

// If the action is invalid, TMSP error will be returned.
func ExecuteAction(state *State, action Action, isCheckTx bool) (res tmsp.Result) {

	chainID := state.GetChainID()

	// Validate Input Basic
	res = action.Input.ValidateBasic()
	if res.IsErr() {
		return res
	}

	var acc *Account

	if action.Type == ActionCreateAccount {
		// Create new account // Must have input pubKey
		var username string
		data, _, err := wire.GetByteSlice(action.Data)
		if err == nil {
			username = string(data)
		} else {
			// No username
		}
		acc = NewAccount(action.Input.PubKey, username)
	} else {
		// Get input account
		acc = state.GetAccount(action.Input.Address)
		if acc == nil {
			return tmsp.ErrBaseUnknownAddress
		}
		if action.Input.PubKey != nil {
			acc.PubKey = action.Input.PubKey
		}
	}

	// Validate input, advanced
	signBytes := action.SignBytes(chainID)
	res = validateInputAdvanced(acc, signBytes, action.Input)
	if res.IsErr() {
		log.Info(Fmt("validateInputAdvanced failed on %X: %v", action.Input.Address, res))
		return res.PrependLog("in validateInputAdvanced()")
	}

	if isCheckTx {
		// CheckTx does not set state
		// Ok, we are done
		return tmsp.OK
	}

	// Increment sequence and create checkpoint
	acc.Sequence += 1
	accCopy := acc.Copy()

	// Run the action.
	cache := state.CacheWrap()
	switch action.Type {
	case ActionCreateAccount:
		res = RunCreateAccount(cache, acc)
	case ActionRemoveAccount:
		res = RunRemoveAccount(cache, acc)
	case ActionSubmitForm:
		res = RunSubmitForm(cache, acc, action.Data)
	default:
		res = tmsp.ErrUnknownRequest.SetLog(
			Fmt("Error unrecognized tx type: %v", action.Type))
	}
	if res.IsOK() {
		//log.Info("Success")
		cache.CacheSync()
	} else {
		log.Info("AppTx failed", "error", res)
		cache.SetAccount(action.Input.Address, accCopy)
	}
	return res
}

//=====================================================================//

func RunCreateAccount(accSetter AccountSetter, acc *Account) tmsp.Result {
	// Set address to new account
	addr := acc.PubKey.Address()
	accSetter.SetAccount(addr, acc)
	return tmsp.OK
}

func RunRemoveAccount(accSetter AccountSetter, acc *Account) tmsp.Result {
	// Set address to nil
	addr := acc.PubKey.Address()
	accSetter.SetAccount(addr, nil)
	return tmsp.OK
}

func RunSubmitForm(state *State, acc *Account, data []byte) (res tmsp.Result) {
	var form Form
	err := wire.ReadBinaryBytes(data, &form)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(Fmt("Failed to decode form data: %v", data))
	}
	formID := form.ID()
	state.Set(formID, data)
	err = state.AddToFilter(formID, form.Issue) // TODO: add location filter
	if err != nil {
		// False positive
	} else {
		log.Info("Added to filter", "filter", form.Issue)
	}
	acc.Addform(form)
	addr := acc.PubKey.Address()
	state.SetAccount(addr, acc)
	return tmsp.OK
}

//=======================================================================================//

func validateInputAdvanced(acc *Account, signBytes []byte, in *ActionInput) (res tmsp.Result) {
	if in == nil {
		// shouldn't happen
	}
	// Check sequence
	seq := acc.Sequence
	if seq+1 != in.Sequence {
		return tmsp.ErrBaseInvalidSequence.AppendLog(
			Fmt("Got %v, expected %v. (acc.seq=%v)", in.Sequence, seq+1, acc.Sequence))
	}
	// Check signature
	if !acc.PubKey.VerifyBytes(signBytes, in.Signature) {
		return tmsp.ErrBaseInvalidSignature.AppendLog(Fmt("SignBytes: %X", signBytes))
	}
	return tmsp.OK
}

//=======================================================================================//

/*
func RunCreateAdminTx(state *State, data []byte) tmsp.Result {

	// Get secret
	secret, _, err := wire.GetByteSlice(data)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error: could not get secret: %v", data))
	}

	// Create keys
	pubKey, privKey := CreateKeys(secret)

	// Create new admin
	newAcc := NewAdmin(pubKey)
	state.SetAccount(pubKey.Address(), newAcc)

	// Return PubKeyBytes
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteByteSlice(pubKey[:], buf, &n, &err)
	wire.WriteByteSlice(privKey[:], buf, &n, &err)
	return tmsp.NewResultOK(buf.Bytes(), "account")
}

func RunRemoveAdminTx(state *State, address []byte) tmsp.Result {
	// Return key so we can remove in AppendTx
	key := AccountKey(address)
	return tmsp.NewResultOK(key, "account")
}

func RunResolveTx(state *State, pubKey crypto.PubKeyEd25519, data []byte) (res tmsp.Result) {
	formID, _, err := wire.GetByteSlice(data)
	if err != nil {
		return tmsp.NewResult(
			ErrDecodeFormID, nil, "")
	}
	value := state.Get(data)
	if len(value) == 0 {
		return tmsp.NewResult(
			ErrFindForm, nil, Fmt("Error cannot find form with ID: %X", formID))
	}
	var form Form
	err = wire.ReadBinaryBytes(value, &form)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error parsing form bytes: %v", err.Error()))
	}
	minuteString := ToTheMinute(TimeString())
	pubKeyString := BytesToHexString(pubKey[:])
	err = (&form).Resolve(minuteString, pubKeyString)
	if err != nil {
		return tmsp.NewResult(
			ErrFormAlreadyResolved, nil, Fmt("Error already resolved form with ID: %v", formID))
	}
	buf, n, err := new(bytes.Buffer), int(0), error(nil)
	wire.WriteBinary(form, buf, &n, &err)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error encoding form with ID: %v", formID))
	}
	state.Set(data, buf.Bytes())

		err = state.AddToFilter(data, "resolved")
		if err != nil {
			// False positive
		}

	data = make([]byte, wire.ByteSliceSize(buf.Bytes())+1)
	bz := data
	bz[0] = ResolveTx
	bz = bz[1:]
	wire.PutByteSlice(bz, buf.Bytes())

	return tmsp.NewResultOK(data, "")
}
*/

/*
	case CreateAdminTx:
		if !acc.PermissionToCreateAdmin() {
			res = tmsp.ErrUnauthorized
		} else {
			res = RunCreateAdminTx(cacheState, action.Data)
		}
	case RemoveAdminTx:
		if !acc.IsAdmin() {
			res = tmsp.ErrUnauthorized
		} else {
			res = RunRemoveAdminTx(cacheState, action.Input.Address)
		}
	case ResolveTx:
		if !acc.PermissionToResolve() {
			res = tmsp.ErrUnauthorized
		} else {
			pubKey := acc.PubKey.(crypto.PubKeyEd25519)
			res = RunResolveTx(cacheState, pubKey, action.Data)
		}
*/
