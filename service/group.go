package service

import (
	"fmt"

	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/httpserve"
)

const (
	// ErrGroupNotFound is returned when a group cannot be found by name
	ErrGroupNotFound = errors.Error("group not found")
)

// Group represents a route group
type Group struct {
	Name string `toml:"name"`
	// Route group
	Group string `toml:"group"`
	// HTTP method
	Method string `toml:"method"`
	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Plugin handlers
	Handlers []string `toml:"handlers"`

	handlers []httpserve.Handler

	g httpserve.Group
}

func (g *Group) init(p plugins) (err error) {
	fmt.Println("Uhh", g, p)
	for _, handlerKey := range g.Handlers {
		var h httpserve.Handler
		if h, err = newPluginHandler(p, handlerKey); err != nil {
			return
		}

		fmt.Println("Appending")
		g.handlers = append(g.handlers, h)
	}

	return
}
