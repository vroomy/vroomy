package vroomy

import (
	"fmt"
	"path"

	"github.com/vroomy/httpserve"
)

func newHandlerGroup(server *httpserve.Serve, prefix string, handlers []httpserve.Handler) *handlerGroup {
	return &handlerGroup{server: server, prefix: prefix, handlers: copySlice(handlers)}
}

// handlerGroup owns middleware at each composition boundary. Register flattened
// routes on the root Serve to avoid httpserve's nested slice aliasing.
type handlerGroup struct {
	server   *httpserve.Serve
	prefix   string
	handlers []httpserve.Handler
}

func (g *handlerGroup) GET(route string, handlers ...httpserve.Handler) error {
	return g.Handle("GET", route, handlers...)
}

func (g *handlerGroup) PUT(route string, handlers ...httpserve.Handler) error {
	return g.Handle("PUT", route, handlers...)
}

func (g *handlerGroup) POST(route string, handlers ...httpserve.Handler) error {
	return g.Handle("POST", route, handlers...)
}

func (g *handlerGroup) DELETE(route string, handlers ...httpserve.Handler) error {
	return g.Handle("DELETE", route, handlers...)
}

func (g *handlerGroup) OPTIONS(route string, handlers ...httpserve.Handler) error {
	return g.Handle("OPTIONS", route, handlers...)
}

func (g *handlerGroup) Handle(method, route string, handlers ...httpserve.Handler) error {
	route = g.joinPath(route)
	if route == "" {
		return fmt.Errorf("route path must be nonempty")
	}

	owned := append(copySlice(g.handlers), handlers...)
	for _, handler := range owned {
		if handler == nil {
			return fmt.Errorf("route %q contains a nil handler: %w", route, ErrInvalidPluginHandler)
		}
	}

	return g.server.Handle(method, route, owned...)
}

func (g *handlerGroup) Group(route string, handlers ...httpserve.Handler) httpserve.Group {
	owned := append(copySlice(g.handlers), handlers...)
	return newHandlerGroup(g.server, g.joinPath(route), owned)
}

func (g *handlerGroup) joinPath(route string) string {
	if g.prefix != "" {
		return path.Join(g.prefix, route)
	}

	return route
}
