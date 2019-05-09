package plugins

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/Hatch1fy/errors"
)

func parseKey(key string) (newKey, alias string) {
	spl := strings.Split(key, " as ")
	// Set key as the first part of the split
	newKey = spl[0]
	// Check to see if an alias was provided
	if len(spl) > 1 {
		// Alias was provided, set the alias value
		alias = spl[1]
	}

	return
}

func gitPull(gitURL string) (err error) {
	goDir := getGitDir(gitURL)
	gitpull := exec.Command("git", "-C", goDir, "pull")
	gitpull.Stdin = os.Stdin

	outBuf := bytes.NewBuffer(nil)
	gitpull.Stdout = outBuf

	errBuf := bytes.NewBuffer(nil)
	gitpull.Stderr = errBuf

	if err = gitpull.Run(); err != nil {
		return errors.Error(errBuf.String())
	}

	outStr := outBuf.String()
	if strings.Index(outStr, "Already up to date.") == 0 {
		return
	}

	fmt.Println(outStr)
	return
}

func goGet(gitURL string, update bool) (err error) {
	args := []string{"get", "-u", "-v", "-buildmode", "plugin", gitURL}
	if !update {
		args = append(args[:1], args[2:]...)
	}

	goget := exec.Command("go", args...)
	goget.Stdin = os.Stdin
	goget.Stdout = os.Stdout

	errBuf := bytes.NewBuffer(nil)
	goget.Stderr = errBuf

	if err = goget.Run(); err != nil {
		return errors.Error(errBuf.String())
	}

	return
}

func goBuild(gitURL, filename string) (err error) {
	goDir := getGoDir(gitURL)
	gobuild := exec.Command("go", "build", "--buildmode", "plugin", "-o", filename, goDir)
	gobuild.Stdin = os.Stdin
	gobuild.Stdout = os.Stdout
	gobuild.Stderr = os.Stderr

	errBuf := bytes.NewBuffer(nil)
	gobuild.Stderr = errBuf

	if err = gobuild.Run(); err != nil {
		return errors.Error(errBuf.String())
	}

	return
}

func getGoDir(gitURL string) (goDir string) {
	homeDir := os.Getenv("HOME")
	return path.Join(homeDir, "go", "src", gitURL)
}

func getGitDir(gitURL string) (goDir string) {
	homeDir := os.Getenv("HOME")
	spl := strings.Split(gitURL, "/")

	var parts []string
	parts = append(parts, homeDir)
	parts = append(parts, "go")
	parts = append(parts, "src")

	if len(spl) > 0 {
		// Append host
		parts = append(parts, spl[0])
	}

	if len(spl) > 1 {
		// Append git user
		parts = append(parts, spl[1])
	}

	if len(spl) > 2 {
		// Append repo name
		parts = append(parts, spl[2])
	}

	// Append git dir
	parts = append(parts, ".git")

	return path.Join(parts...)
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
	info, err := os.Stat(filename)
	if err != nil {
		return
	}

	// Something exists at the provided filename, if it's not a directory - we're good!
	return !info.IsDir()
}

func getGitPluginKey(gitURL string) (key string, err error) {
	_, key, err = getGitURLParts(gitURL)
	return
}

func getGitURLParts(gitURL string) (gitUser, repoName string, err error) {
	var u *url.URL
	if u, err = url.Parse("http://" + gitURL); err != nil {
		return
	}

	parts := stripEmpty(strings.Split(u.Path, "/"))
	gitUser = parts[0]
	repoName = parts[1]
	return
}

func stripEmpty(ss []string) (out []string) {
	for _, str := range ss {
		if len(str) == 0 {
			continue
		}

		out = append(out, str)
	}

	return
}

func getPluginKey(filename string) (key string) {
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

func isGitReference(handlerKey string) (ok bool) {
	var err error
	_, err = url.Parse("http://" + handlerKey)
	return err == nil
}

func closePlugin(p *plugin.Plugin) (err error) {
	var sym plugin.Symbol
	if sym, err = p.Lookup("Close"); err != nil {
		err = nil
		return
	}

	fn, ok := sym.(func() error)
	if !ok {
		return
	}

	return fn()
}

func wrapProcess(fn func() error, ch chan error) {
	ch <- fn()
}

func waitForProcesses(ch chan error, count int) (err error) {
	var n int
	for err = range ch {
		if err != nil {
			return
		}

		if n++; n == count {
			break
		}
	}

	return
}

func isDoesNotExistError(err error) (ok bool) {
	if err == nil {
		return
	}

	str := err.Error()
	return strings.Index(str, "No such file or directory") > -1
}
