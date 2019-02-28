package plugins

import (
	"fmt"
	"path/filepath"
	"plugin"
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
	p.m = make(map[string]*plugin.Plugin)
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
	m map[string]*plugin.Plugin
}

func (p *Plugins) getPlugin(key string) (alias, filename string, err error) {
	if key, alias = parseKey(key); err != nil {
		return
	}

	switch {
	case filepath.Ext(key) != "":
		if len(alias) == 0 {
			alias = getPluginKey(key)
		}

		filename = key
		return

	case isGitReference(key):
		if len(alias) == 0 {
			if alias, err = getGitPluginKey(key); err != nil {
				return
			}
		}

		// Set filename
		filename = filepath.Join(p.dir, alias+".so")

		// Check to see if current plugin exists
		if doesPluginExist(filename) {
			return
		}

		err = p.gitRetrieve(key, filename)

	default:
		fmt.Println("Not supported?")
	}

	return
}

func (p *Plugins) gitRetrieve(gitURL, filename string) (err error) {
	p.out.Notification("About to get: %v", gitURL)
	if err = goGet(gitURL, false); err != nil {
		return
	}

	p.out.Success("Download of %s complete", gitURL)

	p.out.Notification("About to build: %v", gitURL)
	if err = goBuild(gitURL, filename); err != nil {
		return
	}

	p.out.Success("Build of %s complete", gitURL)
	return
}

// New will load a new plugin by plugin key
// The following formats are accepted as keys:
//	- path/to/file/plugin.so
//	- github.com/username/repository/pluginDir
func (p *Plugins) New(pluginKey string) (key string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var alias, filename string
	if alias, filename, err = p.getPlugin(pluginKey); err != nil {
		return
	}

	if _, ok := p.m[alias]; ok {
		err = ErrPluginKeyExists
		return
	}

	if p.m[alias], err = plugin.Open(filename); err != nil {
		return
	}

	p.out.Success("%s (%s) loaded", alias, pluginKey)
	key = alias
	return
}

// Get will return a plugin by key
func (p *Plugins) Get(key string) (plugin *plugin.Plugin, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var ok bool
	if plugin, ok = p.m[key]; !ok {
		err = fmt.Errorf("Cannot find plugin %s: %v", key, ErrPluginNotLoaded)
		return
	}

	return
}
