package cache

import (
	"errors"
	lib "github.com/zballs/3ii/lib"
	. "github.com/zballs/3ii/types"
	"log"
	"time"
)

type Forms map[string]*Form

type Cache struct {
	unresolved chan Forms
	resolved   chan Forms
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

func (cache *Cache) accessUnresolved() Forms {
	return <-cache.unresolved
}

func (cache *Cache) restoreUnresolved(forms Forms, done chan struct{}) {
	cache.unresolved <- forms
	done <- struct{}{}
}

func (cache *Cache) accessResolved() Forms {
	return <-cache.resolved
}

func (cache *Cache) restoreResolved(forms Forms, done chan struct{}) {
	cache.resolved <- forms
	done <- struct{}{}
}

//===========================================================//

func (cache *Cache) NewForm(id string, form *Form) error {
	var err error = nil
	forms := cache.accessUnresolved()
	if forms[id] != nil {
		err = errors.New(form_already_exists)
	} else {
		forms[id] = form
	}
	done := make(chan struct{}, 1)
	go cache.restoreUnresolved(forms, done)
	select {
	case <-done:
		return err
	}
}

func (cache *Cache) findUnresolved(id string) (*Form, error) {
	forms := cache.accessUnresolved()
	form := forms[id]
	done := make(chan struct{}, 1)
	go cache.restoreUnresolved(forms, done)
	select {
	case <-done:
		if form == nil {
			return nil, errors.New(form_not_found)
		}
		return form, nil
	}
}

func (cache *Cache) findResolved(id string) (*Form, error) {
	forms := cache.accessResolved()
	form := forms[id]
	done := make(chan struct{})
	go cache.restoreResolved(forms, done)
	select {
	case <-done:
		if form == nil {
			return nil, errors.New(form_not_found)
		}
		return form, nil
	}
}

func (cache *Cache) FindForm(id string) (*Form, error) {
	form1, err1 := cache.findUnresolved(id)
	form2, err2 := cache.findResolved(id)
	if form1 != nil && err1 == nil && form2 == nil && err2 != nil {
		return form1, nil
	} else if form2 != nil && err2 == nil && form1 == nil && err1 != nil {
		return form2, nil
	}
	return nil, errors.New(form_not_found)
}

func (cache *Cache) removeResolved(id string) {
	forms := cache.accessResolved()
	form := forms[id]
	if form != nil {
		delete(forms, id)
	}
	done := make(chan struct{}, 1)
	go cache.restoreResolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) removeUnresolved(id string) {
	forms := cache.accessUnresolved()
	form := forms[id]
	if form != nil {
		delete(forms, id)
	}
	done := make(chan struct{}, 1)
	go cache.restoreResolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) RemoveForm(id string) {
	done := make(chan struct{}, 1)
	go func() {
		cache.removeResolved(id)
		cache.removeUnresolved(id)
		done <- struct{}{}
	}()
	select {
	case <-done:
		return
	}
}

func (cache *Cache) ResolveForm(id string) error {
	forms1 := cache.accessUnresolved()
	form := forms1[id]
	done := make(chan struct{}, 1)
	if form != nil {
		log.Println(*form)
		go Resolve(time.Now().UTC())(form)
		forms2 := cache.accessResolved()
		forms2[id] = form
		go cache.restoreResolved(forms2, done)
		select {
		case <-done:
			delete(forms1, id)
		}
	}
	go cache.restoreUnresolved(forms1, done)
	select {
	case <-done:
		if form == nil {
			return errors.New(form_not_found)
		}
		return nil
	}
}

func (cache *Cache) searchUnresolved(str string) (formlist Formlist) {
	forms := cache.accessUnresolved()
	for _, form := range forms {
		if MatchForm(str, form) {
			formlist = append(formlist, form)
		}
	}
	done := make(chan struct{}, 1)
	go cache.restoreUnresolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) searchResolved(str string) (formlist Formlist) {
	forms := cache.accessResolved()
	for _, form := range forms {
		if MatchForm(str, form) {
			formlist = append(formlist, form)
		}
	}
	done := make(chan struct{}, 1)
	go cache.restoreResolved(forms, done)
	select {
	case <-done:
		return
	}
}

func (cache *Cache) SearchForms(str string, status string) Formlist {
	if status == "unresolved" {
		return cache.searchUnresolved(str)
	} else if status == "resolved" {
		return cache.searchResolved(str)
	}
	return append(cache.searchUnresolved(str), cache.searchResolved(str)...)
}

// Stats

func (cache *Cache) AvgResponseTime(category string, values ...string) (float64, error) {
	forms := <-cache.resolved
	sum := float64(0)
	count := float64(0)
	if len(values) == 0 {
		for _, form := range forms {
			sum += (*form).ResponseTime
			count += 1
		}
	} else if category == "depts" {
		for _, form := range forms {
			for _, val := range values {
				if lib.SERVICE.ServiceDept((*form).Service) == val {
					sum += (*form).ResponseTime
					count += 1
					break
				}
			}
		}
	} else if category == "services" {
		for _, form := range forms {
			for _, val := range values {
				if (*form).Service == val {
					sum += (*form).ResponseTime
					count += 1
					break
				}
			}
		}
	}
	done := make(chan struct{})
	go cache.restoreResolved(forms, done)
	select {
	case <-done:
		if count > float64(0) {
			return sum / count, nil
		}
		return float64(0), errors.New(zero_forms_found)
	}
}
