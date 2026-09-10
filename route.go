package vroomy

import (
	"fmt"

	"github.com/vroomy/httpserve"
)

// Route represents a listening route
type Route struct {
	// HTTPHandlers are Go-supplied handlers; resolved Handlers are appended to them.
	HTTPHandlers []httpserve.Handler `toml:"-"`

	// Route name/description
	Name string `toml:"name"`
	// Route group
	Group string `toml:"group"`
	// Method selects PUT, POST, DELETE, or OPTIONS case-insensitively; other values use GET.
	Method string `toml:"method"`
	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Target is a legacy file/directory field with no current serving implementation.
	Target string `toml:"target"`
	// Plugin handlers
	Handlers []string `toml:"handlers"`
}

// String will return a formatted version of the route
func (r *Route) String() string {
	return fmt.Sprintf(routeFmt, r.HTTPPath, r.Target, r.Handlers)
}
