package plugins

import (
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/missionMeteora/journaler"
)

func newPlugin(dir, key string, update bool) (pp *Plugin, err error) {
	var p Plugin
	p.importKey = key
	key, p.alias = parseKey(key)
	p.update = update

	switch {
	case filepath.Ext(key) != "":
		if len(p.alias) == 0 {
			p.alias = getPluginKey(key)
		}

		p.filename = key
		return

	case isGitReference(key):
		p.gitURL = key
		if len(p.alias) == 0 {
			if p.alias, err = getGitPluginKey(key); err != nil {
				return
			}
		}

		// Set filename
		p.filename = filepath.Join(dir, p.alias+".so")

	default:
		err = fmt.Errorf("plugin type not supported: %s", key)
		return
	}

	p.out = journaler.New("Plugin", p.alias)
	pp = &p
	return
}

// Plugin represents a plugin entry
type Plugin struct {
	out *journaler.Journaler
	p   *plugin.Plugin

	// Original import key
	importKey string
	// Alias given to plugin (e.g. github.com/user/myplugin would be myplugin)
	alias string
	// The git URL for the plugin
	gitURL string
	// The filename of the plugin's .so file
	filename string

	// Signals if the plugin was loaded with an active update state
	update bool
}

func (p *Plugin) retrieve() (err error) {
	if len(p.gitURL) == 0 {
		return
	}

	if doesPluginExist(p.filename) && !p.update {
		return
	}

	p.out.Notification("About to retrieve")
	if err = gitPull(p.gitURL); !isDoesNotExistError(err) {
		fmt.Println("Error here?", err)
		return
	}

	p.out.Notification("Plugin does not exist, downloading")
	if err = goGet(p.gitURL, false); err != nil {
		return
	}

	p.out.Success("Download complete")
	return
}

func (p *Plugin) build() (err error) {
	if doesPluginExist(p.filename) && !p.update {
		return
	}

	if err = goBuild(p.gitURL, p.filename); err != nil {
		return
	}

	p.out.Success("Build complete")
	return
}

func (p *Plugin) init() (err error) {
	p.p, err = plugin.Open(p.filename)
	return
}
