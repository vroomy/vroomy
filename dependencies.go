package vroomy

import (
	"fmt"
	"reflect"

	"github.com/gdbu/errors"
	"github.com/gdbu/stringset"
)

func makeDependenciesMap(ps map[string]Plugin) (dm dependenciesMap, err error) {
	dm = make(dependenciesMap, len(ps))
	for key, p := range ps {
		if dm[key], err = makeDependencyMap(p); err != nil {
			return nil, fmt.Errorf("inspect plugin %q: %w", key, err)
		}
	}

	return dm, nil
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

	return remaining
}

func (d dependenciesMap) validateRegistration() (err error) {
	for _, dm := range d {
		if err = dm.validateRegistration(d); err != nil {
			return err
		}
	}

	return err
}

func (d dependenciesMap) Load(fn func(pluginKey string, dm dependencyMap) error) (err error) {
	if err = d.validateRegistration(); err != nil {
		return err
	}

	loaded := make(stringset.Map, len(d))
	for len(loaded) < len(d) {
		var passCount int
		for key, dm := range d {
			if loaded.Has(key) {
				continue
			}

			if !dm.isReady(loaded) {
				continue
			}

			if err = fn(key, dm); err != nil {
				return err
			}

			passCount++
			loaded.Set(key)
		}

		if passCount == 0 {
			remaining := d.getRemaining(loaded)
			err = fmt.Errorf("circular import error, affected plugins: %v", remaining)
			return err
		}
	}

	return err
}

func (d dependenciesMap) validateDependency(key string, dm dependencyMap) (err error) {
	if _, ok := dm[key]; ok {
		return fmt.Errorf("self import error: <%s> cannot import itself", key)
	}

	return err
}

func makeDependencyMap(val interface{}) (m dependencyMap, err error) {
	rtype := reflect.TypeOf(val)
	if rtype == nil {
		return nil, fmt.Errorf("cannot inspect nil plugin")
	}

	if rtype.Kind() == reflect.Pointer {
		rtype = rtype.Elem()
	}

	if rtype.Kind() != reflect.Struct {
		return nil, fmt.Errorf("plugin must be a struct, received %v", rtype)
	}

	m = make(dependencyMap)
	if err = appendDependencyMap(m, nil, rtype, make(map[reflect.Type]bool)); err != nil {
		return nil, err
	}

	return m, nil
}

// dependencyMap records tagged field locations for one plugin in the graph.
type dependencyMap map[string][]int

func (d dependencyMap) validateRegistration(dm dependenciesMap) (err error) {
	for key := range d {
		if _, ok := dm[key]; !ok {
			return fmt.Errorf("dependency with key of <%s> not found in dependencies map", key)
		}
	}

	return err
}

func (d dependencyMap) isReady(loaded stringset.Map) (isReady bool) {
	for key := range d {
		if !loaded.Has(key) {
			return false
		}
	}

	return true
}

func appendDependencyMap(m dependencyMap, prefix []int, rtype reflect.Type, visiting map[reflect.Type]bool) error {
	if rtype.Kind() == reflect.Pointer {
		rtype = rtype.Elem()
	}

	if rtype.Kind() != reflect.Struct {
		return nil
	}

	if visiting[rtype] {
		return fmt.Errorf("recursive anonymous embedding in %v", rtype)
	}

	visiting[rtype] = true
	defer delete(visiting, rtype)
	for i := 0; i < rtype.NumField(); i++ {
		field := rtype.Field(i)
		fieldKey := field.Tag.Get("vroomy")
		indices := append(copySlice(prefix), i)
		if fieldKey != "" {
			if field.PkgPath != "" {
				return fmt.Errorf("dependency field %s must be exported: %w", field.Name, ErrNotAddressable)
			}

			if _, exists := m[fieldKey]; exists {
				return fmt.Errorf("duplicate dependency %q; use one shared field", fieldKey)
			}

			m[fieldKey] = indices
			continue
		}

		if field.Anonymous {
			if err := appendDependencyMap(m, indices, field.Type, visiting); err != nil {
				return err
			}
		}
	}

	return nil
}
