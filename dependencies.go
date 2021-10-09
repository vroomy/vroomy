package vroomy

import (
	"fmt"
	"reflect"

	"github.com/gdbu/stringset"
	"github.com/hatchify/errors"
)

func makeDependencyMap(val interface{}) (m dependencyMap) {
	rtype := reflect.TypeOf(val)
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}

	numFields := rtype.NumField()
	m = make(dependencyMap, numFields)
	for i := 0; i < numFields; i++ {
		field := rtype.Field(i)
		fieldValue := field.Tag.Get("vroomy")
		if fieldValue == "" {
			continue
		}

		m[fieldValue] = i
	}

	return
}

func makeDependenciesMap(ps map[string]Plugin) (dm dependenciesMap) {
	dm = make(dependenciesMap, len(ps))
	for key, p := range ps {
		dm[key] = makeDependencyMap(p)
	}

	return
}

type dependenciesMap map[string]dependencyMap

func (d dependenciesMap) Validate() (err error) {
	var errs errors.ErrorList
	for key, dm := range d {
		errs.Push(d.validateDependency(key, dm))
	}

	errs.Push(d.Load(func(_ string, _ dependencyMap) error { return nil }))
	return errs.Err()
}

func (d dependenciesMap) getRemaining(m stringset.Map) (remaining []string) {
	remaining = make([]string, 0, len(d)-len(m))
	for k := range d {
		if m.Has(k) {
			continue
		}

		remaining = append(remaining, k)
	}

	return
}

func (d dependenciesMap) validateRegistration() (err error) {
	for _, dm := range d {
		if err = dm.validateRegistration(d); err != nil {
			return
		}
	}

	return
}

func (d dependenciesMap) Load(fn func(pluginKey string, dm dependencyMap) error) (err error) {
	if err = d.validateRegistration(); err != nil {
		return
	}

	loaded := make(stringset.Map, len(d))
	for len(loaded) < len(d) {
		var passCount int
		for key, dm := range d {
			if !dm.isReady(loaded) {
				continue
			}

			if err = fn(key, dm); err != nil {
				return
			}

			passCount++
			loaded.Set(key)
		}

		if passCount == 0 {
			remaining := d.getRemaining(loaded)
			err = fmt.Errorf("circular import error, affected plugins: %v", remaining)
			return
		}
	}

	return
}

func (d dependenciesMap) validateDependency(key string, dm dependencyMap) (err error) {
	if _, ok := dm[key]; ok {
		return fmt.Errorf("self import error: <%s> cannot import itself", key)
	}

	return
}

type dependencyMap map[string]int

func (d dependencyMap) validateRegistration(dm dependenciesMap) (err error) {
	for key := range d {
		if _, ok := dm[key]; !ok {
			return fmt.Errorf("dependency with key of <%s> not found in dependencies map", key)
		}
	}

	return
}

func (d dependencyMap) isReady(loaded stringset.Map) (isReady bool) {
	for key := range loaded {
		if !loaded.Has(key) {
			return false
		}
	}

	return true
}
