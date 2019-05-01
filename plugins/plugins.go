package plugins

import (
	"fmt"
	"plugin"
	"reflect"
	"sync"

	"github.com/Hatch1fy/errors"
	"github.com/missionMeteora/journaler"
)

const (
	// ErrExpectedEndParen is returned when an ending parenthesis is missing
	ErrExpectedEndParen = errors.Error("expected ending parenthesis")
	// ErrInvalidDir is returned when a directory is empty
	ErrInvalidDir = errors.Error("invalid directory, cannot be empty")
	// ErrPluginKeyExists is returned when a plugin cannot be added because it already exists
	ErrPluginKeyExists = errors.Error("plugin cannot be added, key already exists")
	// ErrPluginNotLoaded is returned when a plugin namespace is provided that has not been loaded
	ErrPluginNotLoaded = errors.Error("plugin with that key has not been loaded")
	// ErrNotAddressable is returned when a non-addressable value is provided
	ErrNotAddressable = errors.Error("provided backend must be addressable")
)

// New will return a new instance of plugins
func New(dir string) (pp *Plugins, err error) {
	if len(dir) == 0 {
		err = ErrInvalidDir
		return
	}

	var p Plugins
	p.out = journaler.New("Plugins")
	p.dir = dir
	p.ps = make(pluginslice, 0, 4)
	pp = &p
	return
}

// Plugins manages loaded plugins
type Plugins struct {
	mu  sync.RWMutex
	out *journaler.Journaler

	// Root directory
	dir string

	// Internal plugin store (by key)
	ps pluginslice

	closed bool
}

// New will load a new plugin by plugin key
// The following formats are accepted as keys:
//	- path/to/file/plugin.so
//	- github.com/username/repository/pluginDir
func (p *Plugins) New(pluginKey string, update bool) (key string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var pi *Plugin
	if pi, err = newPlugin(p.dir, pluginKey, update); err != nil {
		return
	}

	if p.ps, err = p.ps.append(pi); err != nil {
		return
	}

	key = pi.alias
	return
}

// Initialize will initialize all loaded plugins
func (p *Plugins) Initialize() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.out.Notification("Retrieving plugins")
	errs := make(chan error, len(p.ps))
	for _, pi := range p.ps {
		go wrapProcess(pi.retrieve, errs)
	}

	if err = waitForProcesses(errs, len(p.ps)); err != nil {
		return
	}

	p.out.Notification("Building plugins")

	for _, pi := range p.ps {
		if err = pi.build(); err != nil {
			return
		}
	}

	p.out.Notification("Initializing plugins")

	for _, pi := range p.ps {
		if err = pi.init(); err != nil {
			return
		}

		p.out.Success("Initialized %s (%s)", pi.alias, pi.filename)
	}

	return
}

// Get will return a plugin by key
func (p *Plugins) Get(key string) (plugin *plugin.Plugin, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var (
		pi *Plugin
		ok bool
	)

	if pi, ok = p.ps.get(key); !ok {
		err = fmt.Errorf("Cannot find plugin %s: %v", key, ErrPluginNotLoaded)
		return
	}

	plugin = pi.p
	return
}

// Backend will associated the backend of the requested key
func (p *Plugins) Backend(key string, backend interface{}) (err error) {
	var pi *plugin.Plugin
	if pi, err = p.Get(key); err != nil {
		return
	}

	var sym plugin.Symbol
	if sym, err = pi.Lookup("Backend"); err != nil {
		return
	}

	fn, ok := sym.(func() interface{})
	if !ok {
		return fmt.Errorf("invalid symbol, expected func() interface{} and received %v", reflect.TypeOf(sym))
	}

	refVal := reflect.ValueOf(backend)
	elem := refVal.Elem()
	if !elem.CanSet() {
		return ErrNotAddressable
	}

	beVal := reflect.ValueOf(fn())

	if elem.Type() != beVal.Type() {
		return fmt.Errorf("invalid type, expected %v and received %v", elem.Type(), beVal.Type())
	}

	elem.Set(beVal)
	return
}

// Close will close plugins
func (p *Plugins) Close() (err error) {
	p.mu.Lock()
	p.mu.Unlock()
	if p.closed {
		return errors.ErrIsClosed
	}

	var errs errors.ErrorList
	p.out.Notification("Closing plugins")
	for _, pi := range p.ps {
		if err = closePlugin(pi.p); err != nil {
			errs.Push(fmt.Errorf("error closing %s (%s): %v", pi.alias, pi.filename, err))
			continue
		}

		p.out.Success("Closed %s", pi.alias)
	}

	p.closed = true
	return errs.Err()
}

type backendFn func() interface{}
