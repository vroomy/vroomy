package vroomy

import (
	"github.com/gdbu/errors"
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
	// Method is decoded but does not restrict the group's routes.
	Method string `toml:"method"`
	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Plugin handlers
	Handlers []string `toml:"handlers"`

	// HTTPHandlers are Go-supplied handlers; resolved Handlers are appended to them.
	HTTPHandlers []httpserve.Handler `toml:"-"`

	// G is populated during group initialization.
	G httpserve.Group `toml:"-"`
}
