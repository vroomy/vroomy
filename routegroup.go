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
	// Group names the parent group; an empty value selects the root.
	Group string `toml:"group"`
	// Method is decoded but does not restrict the group's routes.
	Method string `toml:"method"`
	// HTTPPath is the group's path prefix.
	HTTPPath string `toml:"httpPath"`
	// Handlers lists plugin method references inherited by descendant routes/groups.
	Handlers []string `toml:"handlers"`

	// HTTPHandlers are Go-supplied handlers; resolved Handlers are appended to them.
	HTTPHandlers []httpserve.Handler `toml:"-"`

	// G is populated during group initialization.
	G httpserve.Group `toml:"-"`
}
