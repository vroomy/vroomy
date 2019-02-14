package service

import (
	"path/filepath"
	"plugin"
	"strings"

	"github.com/Hatch1fy/errors"
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

type plugins map[string]*plugin.Plugin

func getPluginKey(filename string) (key string, err error) {
	base := filepath.Base(filename)
	spl := strings.Split(base, ".")
	key = spl[0]
	return
}
