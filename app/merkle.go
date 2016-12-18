package app

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	. "github.com/zballs/comit/util"
)

const (
	ErrValueNotFound = 10000
)

type MerkleApp struct {
	tree *merkle.IAVLTree
}

func NewMerkleApp() *MerkleApp {
	tree := merkle.NewIAVLTree(0, nil)
	return &MerkleApp{tree}
}

func (merk *MerkleApp) Info() string {
	return Fmt("size:%v", merk.tree.Size())
}

func (merk *MerkleApp) SetOption(key string, value string) (log string) {
	return "No options are supported yet"
}

func (merk *MerkleApp) AppendTx(tx []byte) tmsp.Result {
	if len(tx) == 0 {
		return tmsp.ErrEncodingError.SetLog("Tx length must be greater than zero")
	}
	typeByte := tx[0]
	tx = tx[1:]
	key, n, err := wire.GetByteSlice(tx)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(Fmt("Error getting key: %v", err.Error()))
	}
	tx = tx[n:]
	switch typeByte {
	case 0x01: // Create account, submit form
		value, n, err := wire.GetByteSlice(tx)
		if err != nil {
			return tmsp.ErrEncodingError.SetLog(Fmt("Error getting value: %v", err.Error()))
		}
		tx = tx[n:]
		if len(tx) != 0 {
			return tmsp.ErrEncodingError.SetLog("Got bytes left over")
		}
		merk.tree.Set(key, value)
	case 0x02: // Remove account, resolve form?
		if len(tx) != 0 {
			return tmsp.ErrEncodingError.SetLog("Got bytes left over")
		}
		merk.tree.Remove(key)
	default:
		return tmsp.ErrUnknownRequest.SetLog(Fmt("Unexpected AppendTx type byte %X", typeByte))
	}
	return tmsp.OK
}

func (merk *MerkleApp) CheckTx(tx []byte) tmsp.Result {
	if len(tx) == 0 {
		return tmsp.ErrEncodingError.SetLog("Tx length must be greater than zero")
	}
	_, n, err := wire.GetByteSlice(tx)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(Fmt("Error getting key: %v", err.Error()))
	}
	tx = tx[n:]
	_, n, err = wire.GetByteSlice(tx)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(Fmt("Error getting value: %v", err.Error()))
	}
	tx = tx[n:]
	if len(tx) != 0 {
		return tmsp.ErrEncodingError.SetLog("Got bytes left over")
	}
	return tmsp.OK
}

func (merk *MerkleApp) Commit() tmsp.Result {
	if merk.tree.Size() == 0 {
		return tmsp.NewResultOK(nil, "Empty hash for empty tree")
	}
	hash := merk.tree.Hash()
	return tmsp.NewResultOK(hash, "")
}

func (merk *MerkleApp) Query(query []byte) tmsp.Result {
	if len(query) == 0 {
		return tmsp.ErrEncodingError.SetLog("Query cannot be zero length")
	}
	queryType := query[0]
	switch queryType {
	case QueryValue:
		query = query[1:]
		key, n, err := wire.GetByteSlice(query)
		if err != nil {
			return tmsp.ErrEncodingError.SetLog(Fmt("Error getting key: %v", err.Error()))
		}
		query = query[n:]
		if len(query) != 0 {
			return tmsp.ErrEncodingError.SetLog("Got bytes left over")
		}
		_, value, _ := merk.tree.Get(key)
		if len(value) == 0 {
			return tmsp.NewResult(
				ErrValueNotFound, nil, Fmt("Error no value found for query: %v", query))
		}
		return tmsp.NewResultOK(value, "")
	case QueryIndex:
		query = query[1:]
		index, n, err := wire.GetVarint(query)
		if err != nil {
			return tmsp.ErrEncodingError.SetLog(Fmt("Error getting index: %v", err.Error()))
		}
		query = query[n:]
		if len(query) != 0 {
			return tmsp.ErrEncodingError.SetLog(Fmt("Got bytes left over"))
		}
		key, _ := merk.tree.GetByIndex(index)
		return tmsp.NewResultOK(key, "")
	case QuerySize:
		size := merk.tree.Size()
		data := wire.BinaryBytes(size)
		return tmsp.NewResultOK(data, "")
	case QueryProof:
		query = query[1:]
		key, n, err := wire.GetByteSlice(query)
		if err != nil {
			return tmsp.ErrEncodingError.SetLog(Fmt("Error getting key: %v", err.Error()))
		}
		query = query[n:]
		if len(query) != 0 {
			return tmsp.ErrEncodingError.SetLog("Got bytes left over")
		}
		proof := merk.tree.ConstructProof(key)
		data := wire.BinaryBytes(*proof)
		return tmsp.NewResultOK(data, "")
	default:
		return tmsp.ErrUnknownRequest.SetLog(Fmt("Unexpected Query type byte %X", queryType))
	}
}
