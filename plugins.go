package vroomy

import (
	"fmt"
	"sync"

	"github.com/gdbu/queue"
	"github.com/gdbu/scribe"

	"github.com/hatchify/errors"
)

var p = newPlugins()

func newPlugins() *Plugins {
	var p Plugins
	p.out = scribe.New("Plugins")
	p.pm = make(map[string]Plugin)
	return &p
}

// Plugins manages loaded plugins
type Plugins struct {
	mu  sync.RWMutex
	out *scribe.Scribe

	pm map[string]Plugin

	closed bool
}

// New will load a new plugin by plugin key
// The following formats are accepted as keys:
//   - path/to/file/plugin.so
//   - github.com/username/repository/pluginDir
func (p *Plugins) Register(key string, pi Plugin) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		err = errors.ErrIsClosed
		return
	}

	if _, ok := p.pm[key]; ok {
		return fmt.Errorf("plugin with the key of <%s> has already been loaded", key)
	}

	p.pm[key] = pi
	return
}

// Get will get a plugin by it's key
func (p *Plugins) Get(key string) (pi Plugin, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		err = errors.ErrIsClosed
		return
	}

	var ok bool
	if pi, ok = p.pm[key]; !ok {
		err = fmt.Errorf("plugin with key of <%s> has not been registered", key)
		return
	}

	return
}

func (p *Plugins) Loaded() (pm map[string]Plugin) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pm = make(map[string]Plugin, len(p.pm))
	for key, val := range p.pm {
		pm[key] = val
	}

	return
}

// Test will test all of the plugins
func (p *Plugins) Test() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//for _, pi := range p.pm {
	// TODO: Resolve test stuff here
	//if err = pi.test(); err != nil {
	//	return
	//}
	//}

	return errors.Error("testing has not yet been implemented")

}

// TestAsync will test all of the plugins asynchronously
func (p *Plugins) TestAsync(q *queue.Queue) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//var wg sync.WaitGroup
	//wg.Add(len(p.pm))
	//
	//var errs errors.ErrorList
	//for _, pi := range p.pm {
	//	q.New(func(pi Plugin) func() {
	//		return func() {
	//			defer wg.Done()
	//			// Fix test stuff here
	//		}
	//	}(pi))
	//}
	//
	//wg.Wait()
	//
	//return errs.Err()
	return errors.Error("testing has not yet been implemented")
}

// Close will close plugins
func (p *Plugins) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.ErrIsClosed
	}

	var errs errors.ErrorList
	p.out.Notification("Closing plugins")
	for key, pi := range p.pm {
		if err = pi.Close(); err != nil {
			errs.Push(fmt.Errorf("error closing %s: %v", key, err))
			continue
		}

		p.out.Successf("Closed %s", key)
	}

	p.closed = true
	return errs.Err()
}
