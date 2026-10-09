package vroomy

import (
	stderrors "errors"
	"fmt"
	"log"
	"reflect"
	"strings"
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
	inUse  bool
}

// Register records a plugin under a unique key without initializing it.
func (p *Plugins) Register(key string, pi Plugin) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		err = errors.ErrIsClosed
		return err
	}

	if p.inUse {
		return ErrPluginInUse
	}

	if _, ok := p.pm[key]; ok {
		return fmt.Errorf("plugin with the key of <%s> has already been loaded: %w", key, ErrPluginKeyExists)
	}

	if key == "" || strings.ContainsAny(key, ".()") {
		return fmt.Errorf("invalid plugin key %q", key)
	}

	if err = validatePlugin(pi); err != nil {
		return fmt.Errorf("register plugin %q: %w", key, err)
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
		err = fmt.Errorf("plugin with key of <%s> has not been registered: %w", key, ErrPluginNotLoaded)
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

	if p.inUse {
		return ErrPluginInUse
	}

	var errs []error
	log.Println("Vroomy.Plugins: Closing plugins")
	for key, pi := range p.pm {
		if err = pi.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close registered plugin %q: %w", key, err))
			continue
		}

		log.Printf("Vroomy.Plugins: Closed %s\n", key)
	}

	p.closed = true
	return stderrors.Join(errs...)
}

// validatePlugin protects reflection and promoted lifecycle methods from invalid values.
func validatePlugin(pi Plugin) error {
	value := reflect.ValueOf(pi)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("plugin must be a non-nil pointer to a struct")
	}

	return validatePluginFields(value.Elem(), make(map[reflect.Type]bool))
}

func validatePluginFields(value reflect.Value, visiting map[reflect.Type]bool) error {
	pluginType := value.Type()
	if visiting[pluginType] {
		return fmt.Errorf("recursive anonymous embedding in %v", pluginType)
	}

	visiting[pluginType] = true
	defer delete(visiting, pluginType)
	for i := 0; i < value.NumField(); i++ {
		field := pluginType.Field(i)
		fieldValue := value.Field(i)
		if field.Anonymous && isNilValue(fieldValue) {
			return fmt.Errorf("anonymous field %s must be non-nil", field.Name)
		}

		if field.Tag.Get("vroomy") != "" {
			if !fieldValue.CanSet() {
				return fmt.Errorf("dependency field %s must be exported and settable: %w", field.Name, ErrNotAddressable)
			}

			continue
		}

		if !field.Anonymous {
			continue
		}

		if fieldValue.Kind() == reflect.Pointer {
			if fieldValue.IsNil() {
				return fmt.Errorf("anonymous field %s must be non-nil", field.Name)
			}

			fieldValue = fieldValue.Elem()
		}

		if fieldValue.Kind() == reflect.Struct {
			if fieldErr := validatePluginFields(fieldValue, visiting); fieldErr != nil {
				return fmt.Errorf("anonymous field %s: %w", field.Name, fieldErr)
			}
		}
	}

	return nil
}

// acquire gives one service exclusive lifecycle ownership of the registered instances.
func (p *Plugins) acquire() (map[string]Plugin, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.ErrIsClosed
	}

	if p.inUse {
		return nil, ErrPluginInUse
	}

	plugins := make(map[string]Plugin, len(p.pm))
	for key, plugin := range p.pm {
		plugins[key] = plugin
	}

	p.inUse = true
	return plugins, nil
}

func (p *Plugins) release() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inUse = false
}
