package plugins

import (
	"plugin"
)

// Plugin represents a plugin entry
type Plugin struct {
	alias    string
	filename string

	p *plugin.Plugin
}

func (p *Plugin) init() (err error) {
	p.p, err = plugin.Open(p.filename)
	return
}
