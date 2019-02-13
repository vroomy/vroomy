package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hatch1fy/fileserver"
	"github.com/Hatch1fy/httpserve"
)

// Route represents a listening route
type Route struct {
	fs *fileserver.FileServer

	// HTTP root, used to determine file key
	root string
	// Key of the target file
	// Note: This is only used when the target is a file rather than a directory
	key string

	// HTTP path
	HTTPPath string `toml:"httpPath"`
	// Directory or file to serve
	Target string `toml:"target"`
}

// String will return a formatted version of the route
func (r *Route) String() string {
	return fmt.Sprintf(routeFmt, r.HTTPPath, r.Target)
}

func (r *Route) init() (err error) {
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
