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
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func LoadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
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

func RegisterTemplates(ts ...string) Templates {
	var tmpl = Templates{}
	for _, t := range ts {
		tmpl[t] = template.Must(template.ParseFiles(t, "base.html"))
	}
	return tmpl
}

func RenderTemplate(wr http.ResponseWriter, tmpl Templates, t string, pg *Page) {
	tmpl[t].ExecuteTemplate(wr, "base", &pg)
}

// Handler

func SubmitHandler(wr http.ResponseWriter, req *http.Request, tmpl Templates) {
	pg, _ := LoadPage(string(req.URL.Path[1:]))
	RenderTemplate(wr, tmpl, "submit.html", pg)
}
