package app

import (
	"errors"
	// "log"
	"time"
)

type Forms map[string]*Form

type Cache struct {
	Unresolved chan Forms
	Resolved   chan Forms
}

func CreateCache() *Cache {
	unresolved := make(chan Forms, 1)
	resolved := make(chan Forms, 1)
	done := make(chan struct{}, 1)
	go func() {
		unresolved <- Forms{}
		resolved <- Forms{}
		done <- struct{}{}
	}()
	select {
	case <-done:
		return &Cache{unresolved, resolved}
	}
}

func (cache *Cache) AccessUnresolved() Forms {
	return <-cache.Unresolved
}

func (cache *Cache) ReturnUnresolved(forms Forms, done chan struct{}) {
	cache.Unresolved <- forms
	done <- struct{}{}
}

func (cache *Cache) AccessResolved() Forms {
	return <-cache.Resolved
}

func (cache *Cache) ReturnResolved(forms Forms, done chan struct{}) {
	cache.Resolved <- forms
	done <- struct{}{}
}

func (cache *Cache) NewForm(id string, form *Form) error {
	var err error = nil
	forms := cache.AccessUnresolved()
	if forms[id] != nil {
		err = errors.New("form with ID already exists")
	} else {
		forms[id] = form
	}
	done := make(chan struct{}, 1)
	go cache.ReturnUnresolved(forms, done)
	select {
	case <-done:
		return err
	}
}

func (cache *Cache) ResolveForm(id string) {
	forms1 := cache.AccessUnresolved()
	form := forms1[id]
	go Resolve(time.Now())(form)
	forms2 := cache.AccessResolved()
	forms2[id] = form
	done := make(chan struct{}, 1)
	go cache.ReturnResolved(forms2, done)
	select {
	case <-done:
		delete(forms1, id)
		go cache.ReturnUnresolved(forms1, done)
		select {
		case <-done:
			return
		}
	}
}

func (cache *Cache) QueryUnresolved(id string) (form *Form, err error) {
	forms := cache.AccessUnresolved()
	form = forms[id]
	if form == nil {
		err = errors.New("unresolved form with ID does not exist")
	}
	done := make(chan struct{}, 1)
	go cache.ReturnUnresolved(forms, done)
	select {
	case <-done:
		return form, err
	}
}

func (cache *Cache) QueryResolved(id string) (form *Form, err error) {
	forms := cache.AccessResolved()
	form = forms[id]
	if form == nil {
		err = errors.New("resolved form with ID does not exist")
	}
	done := make(chan struct{})
	go cache.ReturnResolved(forms, done)
	select {
	case <-done:
		return form, err
	}
}

func (cache *Cache) QueryForm(id string) (*Form, error) {
	form1, err1 := cache.QueryUnresolved(id)
	form2, err2 := cache.QueryResolved(id)
	if form1 != nil && err1 == nil && form2 == nil && err2 != nil {
		return form1, nil
	} else if form2 != nil && err2 == nil && form1 == nil && err1 != nil {
		return form2, nil
	}
	return nil, errors.New("form with ID does not exist")
}

// Stats

func (cache *Cache) AvgResponseTime() float64 {
	forms := <-cache.Resolved
	sum := float64(0)
	count := float64(0)
	for _, form := range forms {
		sum += (*form).ResponseTime
		count += 1
	}
	done := make(chan struct{})
	go cache.ReturnResolved(forms, done)
	select {
	case <-done:
		return sum / count
	}
}
