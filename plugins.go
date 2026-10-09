package vroomy

import (
	"fmt"
	"log"
	"sync"

	"github.com/gdbu/errors"
	"github.com/gdbu/queue"
)

var p = newPlugins()

func newPlugins() *Plugins {
	var p Plugins
	p.pm = make(map[string]Plugin)
	return &p
}

// Plugins manages a plugin registry. Services use the package-level registry via
// Register; the zero value is not ready for registration.
type Plugins struct {
	mu sync.RWMutex

	pm map[string]Plugin

	closed bool
}

// Register records a plugin under a unique key without initializing it.
func (p *Plugins) Register(key string, pi Plugin) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		err = errors.ErrIsClosed
		return err
	}

	if _, ok := p.pm[key]; ok {
		return fmt.Errorf("plugin with the key of <%s> has already been loaded", key)
	}

	p.pm[key] = pi
	return err
}

// Get returns a plugin by its registration key.
func (p *Plugins) Get(key string) (pi Plugin, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		err = errors.ErrIsClosed
		return pi, err
	}

	var ok bool
	if pi, ok = p.pm[key]; !ok {
		err = fmt.Errorf("plugin with key of <%s> has not been registered", key)
		return pi, err
	}

	return pi, err
}

// Loaded returns a copy of the registry map, sharing the same plugin instances.
func (p *Plugins) Loaded() (pm map[string]Plugin) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pm = make(map[string]Plugin, len(p.pm))
	for key, val := range p.pm {
		pm[key] = val
	}

	return pm
}

// Test is unimplemented and always returns an error. Use go test for repository tests.
func (p *Plugins) Test() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return errors.Error("testing has not yet been implemented")
}

// TestAsync is unimplemented and always returns an error; q is not used.
func (p *Plugins) TestAsync(q *queue.Queue) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return errors.Error("testing has not yet been implemented")
}

// Close closes registered plugins in unspecified order and marks the registry closed.
func (p *Plugins) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.ErrIsClosed
	}

	var errs errors.ErrorList
	log.Println("Vroomy.Plugins: Closing plugins")
	for key, pi := range p.pm {
		if err = pi.Close(); err != nil {
			errs.Push(fmt.Errorf("error closing %s: %v", key, err))
			continue
		}

		log.Printf("Vroomy.Plugins: Closed %s\n", key)
	}

	p.closed = true
	return errs.Err()
}
