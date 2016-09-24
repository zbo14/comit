package app

import (
	. "github.com/tendermint/go-common"
	merkle "github.com/tendermint/go-merkle"
	types "github.com/tendermint/tmsp/types"
	. "github.com/zballs/3ii/accounts"
	. "github.com/zballs/3ii/cache"
	. "github.com/zballs/3ii/types"
	"log"
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
	form, err := MakeForm(string(tx))
	if err != nil {
		log.Println(err.Error())
	}
	log.Println(*form)
	id := FormID(form)
	err = app.cache.NewForm(id, form)
	if err != nil {
		return types.NewResult(types.CodeType_InternalError, nil, err.Error())
	}
	app.state.Set([]byte(id), tx)
	return types.NewResultOK(nil, id)
}

func (app *Application) CheckTx(tx []byte) types.Result {
	_, err := MakeForm(string(tx))
	if err != nil {
		return types.NewResult(types.CodeType_InternalError, nil, err.Error())
	}
	return types.NewResultOK(nil, "")
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
