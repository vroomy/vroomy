package service

import (
	"fmt"
	"path/filepath"

	"github.com/Hatch1fy/errors"
)

const (
	// ErrInvalidRoot is returned whe a root is longer than the request path
	ErrInvalidRoot = errors.Error("invalid root, cannot be longer than request path")
	eerrr          = 1
)

func getKeyFromRequestPath(root, requestPath string) (key string, err error) {
	fmt.Println("Getting key", root)
	fmt.Println(requestPath)
	// Clean request path
	requestPath = filepath.Clean(requestPath)

	if len(root) > len(requestPath) {
		err = ErrInvalidRoot
		return
	}

	key = requestPath[len(root):]
	return
}
