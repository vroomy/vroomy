package service

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/fileserver"
	"github.com/Hatch1fy/httpserve"
	"github.com/vroomy/plugins"
)

const (
	// ErrInvalidPluginHandler is returned when a plugin handler is not valid
	ErrInvalidPluginHandler = errors.Error("plugin handler not valid")
	// ErrExpectedEndParen is returned when an ending parenthesis is missing
	ErrExpectedEndParen = errors.Error("expected ending parenthesis")
)

type route struct {
}

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
	handlers []httpserve.Handler

	// Route group
	Group string `toml:"group"`
	// HTTP method
	Method string `toml:"method"`
	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Directory or file to serve
	Target string `toml:"target"`
	// Plugin handlers
	Handlers []string `toml:"handlers"`
}

// String will return a formatted version of the route
func (r *Route) String() string {
	return fmt.Sprintf(routeFmt, r.HTTPPath, r.Target, r.Handlers)
}

func (r *Route) init(p *plugins.Plugins) (err error) {
	if len(r.Handlers) > 0 {
		if err = r.initPlugins(p); err != nil {
			return
		}

		// Note: We are going to support the ability to serve a target even if we already
		// have handlers specified. This will allow us to add MW logic to file serving routes
		// in addition to our standard plugin routes.
		// TODO: Determine if we want to move file serving to a plugin approach, and remove it
		// from the core vroomy offerings
		if len(r.Target) == 0 {
			// No target is set, bail out now
			return
		}
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

	// Initialize the file server
	if r.fs, err = fileserver.New(target); err != nil {
		return
	}

	// Set root as the target
	r.root, _ = filepath.Split(r.HTTPPath)
	r.handlers = append(r.handlers, r.serveHTTP)
	return
}

func (r *Route) initPlugins(p *plugins.Plugins) (err error) {
	for _, handlerKey := range r.Handlers {
		var h httpserve.Handler
		if h, err = newPluginHandler(p, handlerKey); err != nil {
			return
		}

		r.handlers = append(r.handlers, h)
	}

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

        b, _ := io.Copy(ioutil.Discard, ctx.Request.Body)
        if b == 0 {
            defer ctx.Request.Body.Close()
        }

	return
}
