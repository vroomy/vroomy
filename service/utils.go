package service

import (
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/httpserve"
	"github.com/Hatch1fy/vroomie/plugins"
)

const (
	// ErrInvalidRoot is returned whe a root is longer than the request path
	ErrInvalidRoot = errors.Error("invalid root, cannot be longer than request path")
	eerrr          = 1
)

func getKeyFromRequestPath(root, requestPath string) (key string, err error) {
	// Clean request path
	requestPath = filepath.Clean(requestPath)

	if len(root) > len(requestPath) {
		err = ErrInvalidRoot
		return
	}

	key = requestPath[len(root):]
	return
}

func trimSlash(in string) (out string) {
	if len(in) == 0 {
		return
	}

	if in[len(in)-1] != '/' {
		return in
	}

	return in[:len(in)-1]
}

func getHandlerParts(handlerKey string) (key, handler string, args []string, err error) {
	spl := strings.Split(handlerKey, ".")
	key = spl[0]
	handler = spl[1]

	spl = strings.Split(handler, "(")
	if len(spl) == 1 {
		return
	}

	handler = spl[0]
	argsStr := spl[1]
	if argsStr[len(argsStr)-1] != ')' {
		err = ErrExpectedEndParen
		return
	}

	argsStr = argsStr[:len(argsStr)-1]
	args = strings.Split(argsStr, ",")
	return
}

func newPluginHandler(p *plugins.Plugins, handlerKey string) (h httpserve.Handler, err error) {
	var (
		key     string
		handler string
		args    []string
	)

	if key, handler, args, err = getHandlerParts(handlerKey); err != nil {
		return
	}

	var pp *plugin.Plugin
	if pp, err = p.Get(key); err != nil {
		return
	}

	var sym plugin.Symbol
	if sym, err = pp.Lookup(handler); err != nil {
		return
	}

	switch v := sym.(type) {
	case func(*httpserve.Context) httpserve.Response:
		h = v
	case func(args ...string) (httpserve.Handler, error):
		if h, err = v(args...); err != nil {
			return
		}
	}

	return
}

func initDir(loc string) (err error) {
	if err = os.Mkdir(loc, 0744); err == nil {
		return
	}

	if os.IsExist(err) {
		return nil
	}

	return
}
