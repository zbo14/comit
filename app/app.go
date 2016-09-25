package app

import (
	. "github.com/tendermint/go-common"
	merkle "github.com/tendermint/go-merkle"
	types "github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/accounts"
	. "github.com/zballs/3ii/cache"
	lib "github.com/zballs/3ii/lib"
	. "github.com/zballs/3ii/types"
	util "github.com/zballs/3ii/util"
)

type Application struct {
	state         merkle.Tree
	user_manager  *UserManager
	admin_manager *AdminManager
	cache         *Cache
}

func NewApplication() *Application {
	state := merkle.NewIAVLTree(
		0,
		nil,
	)
	return &Application{
		state:         state,
		user_manager:  CreateUserManager(),
		admin_manager: CreateAdminManager(8),
		cache:         CreateCache(),
	}
}

func (app *Application) UserManager() *UserManager {
	return app.user_manager
}

func (app *Application) AdminManager() *AdminManager {
	return app.admin_manager
}

func (app *Application) Cache() *Cache {
	return app.cache
}

// TMSP requests

func (app *Application) Info() string {
	return Fmt("size:%v", app.state.Size())
}

func (app *Application) SetOption(key string, value string) (log string) {
	return ""
}

func (app *Application) AppendTx(tx []byte) types.Result {
	str := string(tx)
	action := lib.SERVICE.ReadField(str, "action")
	if action == "submit" {
		form, err := MakeForm(str)
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		ID := FormID(form)
		err = app.cache.NewForm(ID, form)
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		app.state.Set([]byte(ID), []byte("submit"))
		newstr := lib.SERVICE.WriteField(ID, "ID") + str
		return types.NewResultOK([]byte(newstr), ID)
	} else if action == "resolve" {
		ID := lib.SERVICE.ReadField(str, "ID")
		pubKeyString := util.ReadPubKeyString(str)
		form, err := app.Cache().ResolveForm(ID)
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		newstr := lib.SERVICE.WriteField(ID, "ID") +
			lib.SERVICE.WriteField("resolve", "action") +
			lib.SERVICE.WriteField(form.Service(), "service") +
			lib.SERVICE.WriteField(form.Address(), "address") +
			util.WritePubKeyString(pubKeyString)
		app.state.Set([]byte(ID), []byte("resolve"))
		return types.NewResultOK([]byte(newstr), "")
	}
	return types.NewResult(types.CodeType_UnknownRequest, nil, "")
}

func (app *Application) CheckTx(tx []byte) types.Result {
	str := string(tx)
	action := lib.SERVICE.ReadField(str, "action")
	if action == "submit" {
		form, err := MakeForm(string(tx))
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		ID := FormID(form)
		err = app.cache.NewForm(ID, form)
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		return types.NewResultOK(nil, "")
	} else if action == "resolve" {
		ID := lib.SERVICE.ReadField(str, "ID")
		_, err := app.Cache().ResolveForm(ID)
		if err != nil {
			return types.NewResult(types.CodeType_InternalError, nil, err.Error())
		}
		return types.NewResultOK(nil, "")
	}
	return types.NewResult(types.CodeType_UnknownRequest, nil, "")
}

func (app *Application) Commit() types.Result {
	hash := app.state.Hash()
	return types.NewResultOK(hash, "")
}

func (app *Application) Query(query []byte) types.Result {
	_, _, exists := app.state.Get(query)
	if exists {
		return types.NewResultOK(nil, "")
	}
	return types.NewResult(types.CodeType_InternalError, nil, "")
}
