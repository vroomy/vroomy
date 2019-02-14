package service

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/fileserver"
	"github.com/Hatch1fy/httpserve"
)

const (
	// ErrPluginNotLoaded is returned when a requested plugin has not been loaded
	ErrPluginNotLoaded = errors.Error("plugin not loaded")
	// ErrInvalidPluginHandler is returned when a plugin handler is not valid
	ErrInvalidPluginHandler = errors.Error("plugin handler not valid")
)

// Route represents a listening route
type Route struct {
	fs *fileserver.FileServer

	// HTTP root, used to determine file key
	root string
	// Key of the target file
	// Note: This is only used when the target is a file rather than a directory
	key string

	// Target plug-in handler
	// Note: This is only used when the target is a plugin handler
	handler httpserve.Handler

	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Directory or file to serve
	Target string `toml:"target"`
	// Plugin handler
	Handler string `toml:"handler"`
}

// String will return a formatted version of the route
func (r *Route) String() string {
	return fmt.Sprintf(routeFmt, r.HTTPPath, r.Target, r.Handler)
}

func (r *Route) init(p plugins) (err error) {
	fmt.Println("Initing", r.Handler)
	if r.Handler != "" {
		return r.initPlugin(p)
	}

	var info os.FileInfo
	target := r.Target
	if info, err = os.Stat(target); err != nil {
		return
	}

	switch mode := info.Mode(); {
	case mode.IsDir():
		// Target is a directory, we're good to go!
	case mode.IsRegular():
		// Target is a file, we must perform some actions
		// Set the file key
		r.key = filepath.Base(target)
		// Truncate the target to represent the directory
		target = filepath.Dir(target)
	}

	fmt.Println("Initializing file server at", target)
	// Initialize the file server
	if r.fs, err = fileserver.New(target); err != nil {
		return
	}

	// Set root as the target
	r.root, _ = filepath.Split(r.HTTPPath)
	r.handler = r.serveHTTP
	return
}

func (r *Route) initPlugin(p plugins) (err error) {
	spl := strings.Split(r.Handler, ".")
	key := spl[0]
	handler := spl[1]

	pp, ok := p[key]
	if !ok {
		return ErrPluginNotLoaded
	}

	var sym plugin.Symbol
	if sym, err = pp.Lookup(handler); err != nil {
		return
	}

	r.handler, ok = sym.(func(*httpserve.Context) httpserve.Response)
	if !ok {
		fmt.Println("Uhh", sym)
		return ErrInvalidPluginHandler
	}

	fmt.Println("Plugin handler set!", key)
	return
}

func (r *Route) getKey(requestPath string) (key string, err error) {
	if len(r.key) > 0 {
		key = r.key
		return
	}

	return getKeyFromRequestPath(r.root, requestPath)
}

func (r *Route) serveHTTP(ctx *httpserve.Context) (res httpserve.Response) {
	var (
		key string
		err error
	)

	if key, err = r.getKey(ctx.Request.URL.Path); err != nil {
		return httpserve.NewTextResponse(400, []byte(err.Error()))
	}

	if err := r.fs.Serve(key, ctx.Writer, ctx.Request); err != nil {
		err = fmt.Errorf("Error serving %s: %v", key, err)
		return httpserve.NewTextResponse(400, []byte(err.Error()))
	}

	return
}
