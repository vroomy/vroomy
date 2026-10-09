package vroomy

import (
	"fmt"

	"github.com/vroomy/httpserve"
)

// routeFmt formats a route for display.
const routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" Plugin Handler: \"%v\" }"

// Route represents a listening route
type Route struct {
	// HTTPHandlers are Go-supplied handlers; resolved Handlers are appended to them.
	HTTPHandlers []httpserve.Handler `toml:"-"`

	// Name describes the route.
	Name string `toml:"name"`
	// Group names the parent group; an empty value selects the root.
	Group string `toml:"group"`
	// Method selects PUT, POST, DELETE, or OPTIONS case-insensitively; other values use GET.
	Method string `toml:"method"`
	// HTTPPath is the route path, joined to a nonempty group prefix.
	HTTPPath string `toml:"httpPath"`
	// Target is a legacy file/directory field with no current serving implementation.
	Target string `toml:"target"`
	// Handlers lists registered plugin method references in execution order.
	Handlers []string `toml:"handlers"`
}

// String will return a formatted version of the route
func (r *Route) String() string {
	return fmt.Sprintf(routeFmt, r.HTTPPath, r.Target, r.Handlers)
}
