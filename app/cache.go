package app

import (
	"errors"
	"time"
)

type Cache struct {
	Forms    chan map[string]*Form
	Resolved chan map[string]*Form
}

func CreateCache() *Cache {
	return &Cache{
		Forms:    make(chan map[string]*Form, 1),
		Resolved: make(chan map[string]*Form, 1),
	}
}

func (cache *Cache) Init(form *Form) {
	cache.Forms <- map[string]*Form{}
	cache.Resolved <- map[string]*Form{}
}

func (cache *Cache) NewForm(id string, form *Form) error {
	forms := <-cache.Forms
	if forms[id] != nil {
		return errors.New("form with ID already exists")
	}
	forms[id] = form
	cache.Forms <- forms
	return nil
}

func (cache *Cache) ResolveForm(id string) {
	forms := <-cache.Forms
	resolved := <-cache.Resolved
	Resolve(time.Now())(forms[id])
	resolved[id] = forms[id]
	cache.Resolved <- resolved
	delete(forms, id)
	cache.Forms <- forms
}

func (cache *Cache) QueryForm(id string) (form *Form) {
	forms := <-cache.Forms
	form = forms[id]
	done := make(chan struct{})
	go func() {
		cache.Forms <- forms
		done <- struct{}{}
	}()
	select {
	case <-done:
		return
	}
}

func (cache *Cache) QueryResolved(id string) (form *Form) {
	resolved := <-cache.Resolved
	form = resolved[id]
	done := make(chan struct{})
	go func() {
		cache.Resolved <- resolved
		done <- struct{}{}
	}()
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
