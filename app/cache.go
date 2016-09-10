package app

import (
	"errors"
	"log"
	"time"
)

type Forms map[string]*Form

type Cache struct {
	Forms    chan Forms
	Resolved chan Forms
}

func CreateCache() *Cache {
	return &Cache{
		Forms:    make(chan Forms, 1),
		Resolved: make(chan Forms, 1),
	}
}

func (cache *Cache) Init(form *Form) {
	cache.Forms <- Forms{}
	cache.Resolved <- Forms{}
}

func (cache *Cache) AccessForms() Forms {
	return <-cache.Forms
}

func (cache *Cache) AccessResolved() Forms {
	return <-cache.Resolved
}

func (cache *Cache) ReturnForms(forms Forms, done chan struct{}) {
	cache.Forms <- forms
	done <- struct{}{}
}

func (cache *Cache) ReturnResolved(resolved Forms, done chan struct{}) {
	cache.Resolved <- resolved
	done <- struct{}{}
}

func (cache *Cache) NewForm(id string, form *Form) error {
	var err error = nil
	forms := cache.AccessForms()
	if forms[id] != nil {
		err = errors.New("form with ID already exists")
	} else {
		forms[id] = form
	}
	log.Println(*form)
	done := make(chan struct{}, 1)
	go cache.ReturnForms(forms, done)
	select {
	case <-done:
		return err
	}
}

func (cache *Cache) ResolveForm(id string) {
	forms := cache.AccessForms()
	resolved := cache.AccessResolved()
	Resolve(time.Now())(forms[id])
	resolved[id] = forms[id]
	cache.Resolved <- resolved
	delete(forms, id)
	cache.Forms <- forms
}

func (cache *Cache) QueryForm(id string) (form *Form) {
	forms := cache.AccessForms()
	form = forms[id]
	done := make(chan struct{}, 1)
	go cache.ReturnForms(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) QueryResolved(id string) (form *Form) {
	resolved := cache.AccessResolved()
	form = resolved[id]
	done := make(chan struct{})
	go cache.ReturnResolved(resolved, done)
	select {
	case <-done:
		return
	}
}

// Stats

func (cache *Cache) AvgResponseTime() float64 {
	resolved := <-cache.Resolved
	sum := float64(0)
	count := float64(0)
	for _, form := range resolved {
		sum += (*form).ResponseTime
		count += 1
	}
	done := make(chan struct{})
	go func() {
		cache.Resolved <- resolved
		done <- struct{}{}
	}()
	select {
	case <-done:
		return sum / count
	}
}
