package service

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
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

func getFilename(dir, key string) (filename string, err error) {
	fmt.Println("Checking filename ", key)
	switch {
	case filepath.Ext(key) != "":
		fmt.Println("SO")
		filename = key
		return

	case isGitReference(key):
		fmt.Println("Git reference!", key)
		var pluginKey string
		if pluginKey, err = getPluginKey(key); err != nil {
			fmt.Println("What?", err)
			return
		}

		if filename = filepath.Join(dir, pluginKey+".so"); doesPluginExist(filename) {
			fmt.Println("Exists!", filename)
			return
		}

		fmt.Println("Need to get from github", key)
		return getFromGithub(dir, key)

	default:
		fmt.Println("Not supported?")
	}

	return
}

func getFromGithub(dir, gitURL string) (filename string, err error) {
	var u *url.URL
	if u, err = url.Parse("http://" + gitURL); err != nil {
		return
	}

	spl := strings.Split(u.Path, "/")
	if len(spl) > 2 {
		spl = spl[:3]
	}

	u.Path = path.Join(spl...)

	downloadURL := u.String()[7:]
	fmt.Println("Download url", downloadURL)

	goget := exec.Command("go", "get", downloadURL)
	goget.Stdin = os.Stdin
	goget.Stdout = os.Stdout
	goget.Stderr = os.Stderr

	fmt.Println("About to download", gitURL)
	if err = goget.Run(); err != nil {
		fmt.Println("Error", err)
		return
	}

	homeDir := os.Getenv("HOME")
	goDir := path.Join(homeDir, "go", "src", gitURL)
	filename = filepath.Join(dir, filepath.Base(gitURL)+".so")

	gobuild := exec.Command("go", "build", "--buildmode", "plugin", "-o", filename, goDir)
	gobuild.Stdin = os.Stdin
	gobuild.Stdout = os.Stdout
	gobuild.Stderr = os.Stderr

	fmt.Println("About to build", gitURL)
	if err = gobuild.Run(); err != nil {
		fmt.Println("Error", err)
		return
	}

	fmt.Println("Yay")
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

func doesPluginExist(filename string) (exists bool) {
	fmt.Println("Check if plugin exists", filename)
	return
}

func getPluginKey(filename string) (key string, err error) {
	base := filepath.Base(filename)
	spl := strings.Split(base, ".")
	key = spl[0]
	return
}

func getKeyFromGitURL(gitURL string) (key string, err error) {
	var u *url.URL
	if u, err = url.Parse("http://" + gitURL); err != nil {
		return
	}

	key = filepath.Base(u.Path)
	return
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

func isGitReference(handlerKey string) (ok bool) {
	var err error
	_, err = url.Parse("http://" + handlerKey)
	return err == nil
}
