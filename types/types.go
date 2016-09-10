package types

import (
	"html/template"
	"io/ioutil"
	"net/http"
)

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) Save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile("pages/"+filename, p.Body, 0600)
}

func LoadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile("pages/" + filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func CreatePages(titles ...string) {
	var pg *Page
	for _, title := range titles {
		pg = &Page{Title: title, Body: []byte("")}
		pg.Save()
	}
}

// File system

type JustFiles struct {
	Fs http.FileSystem
}

type MyFile struct {
	http.File
}

func (js JustFiles) Open(filename string) (http.File, error) {
	f, err := js.Fs.Open(filename)
	if err != nil {
		return nil, err
	}
	return MyFile{f}, nil
}

// Templates

type Templates map[string]*template.Template

var Tmpl = Templates{}

func RegisterTemplates(ts ...string) {
	for _, t := range ts {
		Tmpl[t] = template.Must(template.ParseFiles("templates/"+t, "templates/base.html"))
	}
}

func RenderTemplate(wr http.ResponseWriter, t string, pg *Page) {
	Tmpl[t].ExecuteTemplate(wr, "base", &pg)
}

// Handler

func CreateAccountHandler(wr http.ResponseWriter, req *http.Request) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, "create_account.html", pg)
}

func RemoveAccountHandler(wr http.ResponseWriter, req *http.Request) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, "remove_account.html", pg)
}

func SubmitFormHandler(wr http.ResponseWriter, req *http.Request) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, "submit_form.html", pg)
}

func QueryFormHandler(wr http.ResponseWriter, req *http.Request) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, "query_form.html", pg)
}

func ResolveFormHandler(wr http.ResponseWriter, req *http.Request) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, "resolve_form.html", pg)
}
