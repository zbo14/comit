package actions

import (
	"github.com/zballs/comit/forms"
)

type Error struct {
	Error string `json:"error"`
}

var WriteRequestError = Error{"write request failure"}
var ReadResponseError = Error{"read response failures"}

var InvalidPublicKey = Error{"invalid public key"}
var InvalidPrivateKey = Error{"invalid private key"}
var InvalidFormID = Error{"invalid form ID"}

type CreateAccount struct {
	Status     string `json:"create account"`
	PubKeystr  string `json:"public key"`
	PrivKeystr string `json:"private key"`
}

var CreateAccountFailure = CreateAccount{Status: "failure"}

type RemoveAccount struct {
	Status string `json:"remove account"`
}

var RemoveAccountFailure = RemoveAccount{"failure"}
var RemoveAccountSuccess = RemoveAccount{"success"}

type CreateAdmin struct {
	Status     string `json:"create admin"`
	PubKeystr  string `json:"public key"`
	PrivKeystr string `json:"private key"`
}

var CreateAdminFailure = CreateAdmin{Status: "failure"}

type RemoveAdmin struct {
	Status string `json:"remove admin"`
}

var RemoveAdminFailure = RemoveAdmin{"failure"}
var RemoveAdminSuccess = RemoveAdmin{"success"}

type ConnectMsg struct {
	Status string `json:"connect"`
	Type   string `json:"type"`
}

var ConnectFailure = ConnectMsg{Status: "failure"}
var ConnectConstituent = ConnectMsg{Status: "success", Type: "constituent"}
var ConnectAdmin = ConnectMsg{Status: "success", Type: "admin"}

type SubmitForm struct {
	Status string `json:"submit form"`
	FormID string `json:"form ID"`
}

var SubmitFormFailure = SubmitForm{Status: "failure"}

type ResolveForm struct {
	Status string `json:"resolve form"`
}

var ResolveFormFailure = ResolveForm{Status: "failure"}
var ResolveFormSuccess = ResolveForm{Status: "success"}

type FindForm struct {
	Status string     `json:"find form"`
	Form   forms.Form `json:"form"`
}

var FindFormFailure = FindForm{Status: "failure"}

var select_option = `<option value="%s">%s</option>`
