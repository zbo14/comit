package app

import (
	"errors"
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

func (cache *Cache) RestoreUnresolved(forms Forms, done chan struct{}) {
	cache.Unresolved <- forms
	done <- struct{}{}
}

func (cache *Cache) AccessResolved() Forms {
	return <-cache.Resolved
}

func (cache *Cache) RestoreResolved(forms Forms, done chan struct{}) {
	cache.Resolved <- forms
	done <- struct{}{}
}

func (cache *Cache) NewForm(id string, form *Form) error {
	var err error = nil
	forms := cache.AccessUnresolved()
	if forms[id] != nil {
		err = errors.New(form_already_exists)
	} else {
		forms[id] = form
	}
	done := make(chan struct{}, 1)
	go cache.RestoreUnresolved(forms, done)
	select {
	case <-done:
		return err
	}
}

func (cache *Cache) FindUnresolved(id string) (form *Form, err error) {
	forms := cache.AccessUnresolved()
	form = forms[id]
	if form == nil {
		err = errors.New("unresolved form with ID not found")
	}
	done := make(chan struct{}, 1)
	go cache.RestoreUnresolved(forms, done)
	select {
	case <-done:
		return form, err
	}
}

func (cache *Cache) FindResolved(id string) (form *Form, err error) {
	forms := cache.AccessResolved()
	form = forms[id]
	if form == nil {
		err = errors.New("resolved form with ID not found")
	}
	done := make(chan struct{})
	go cache.RestoreResolved(forms, done)
	select {
	case <-done:
		return form, err
	}
}

func (cache *Cache) FindForm(id string) (*Form, error) {
	form1, err1 := cache.FindUnresolved(id)
	form2, err2 := cache.FindResolved(id)
	if form1 != nil && err1 == nil && form2 == nil && err2 != nil {
		return form1, nil
	} else if form2 != nil && err2 == nil && form1 == nil && err1 != nil {
		return form2, nil
	}
	return nil, errors.New("valid form with ID not found")
}

func (cache *Cache) ResolveForm(id string) error {
	forms1 := cache.AccessUnresolved()
	form := forms1[id]
	done := make(chan struct{}, 1)
	if form != nil {
		go Resolve(time.Now())(form)
		forms2 := cache.AccessResolved()
		forms2[id] = form
		go cache.RestoreResolved(forms2, done)
		select {
		case <-done:
			delete(forms1, id)
		}
	}
	go cache.RestoreUnresolved(forms1, done)
	select {
	case <-done:
		if form == nil {
			return errors.New("unresolved form with ID not found")
		}
		return nil
	}
}

func (cache *Cache) SearchUnresolved(str string) (formlist Formlist) {
	forms := cache.AccessUnresolved()
	for _, form := range forms {
		if MatchForm(str, form) {
			formlist = append(formlist, form)
		}
	}
	done := make(chan struct{}, 1)
	go cache.RestoreUnresolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) SearchResolved(str string) (formlist Formlist) {
	forms := cache.AccessResolved()
	for _, form := range forms {
		if MatchForm(str, form) {
			formlist = append(formlist, form)
		}
	}
	done := make(chan struct{}, 1)
	go cache.RestoreResolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) SearchForms(str string, _status string) Formlist {
	if _status == "unresolved" {
		return cache.SearchUnresolved(str)
	} else if _status == "resolved" {
		return cache.SearchResolved(str)
	}
	return append(cache.SearchUnresolved(str), cache.SearchResolved(str)...)
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
	go cache.RestoreResolved(forms, done)
	select {
	case <-done:
		return sum / count
	}
}
