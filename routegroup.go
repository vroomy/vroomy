package vroomy

import (
	"github.com/hatchify/errors"
	"github.com/vroomy/httpserve"
)

const (
	// ErrGroupNotFound is returned when a group cannot be found by name
	ErrGroupNotFound = errors.Error("group not found")
)

// RouteGroup represents a route group
type RouteGroup struct {
	Name string `toml:"name"`
	// Route group
	Group string `toml:"group"`
	// HTTP method
	Method string `toml:"method"`
	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Plugin handlers
	Handlers []string `toml:"handlers"`

	HTTPHandlers []httpserve.Handler `toml:"-"`

	G httpserve.Group `toml:"-"`
}
