package plugins

import (
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

func goGet(gitURL string, update bool) (err error) {
	var downloadURL string
	if downloadURL, err = getGitDownloadURL(gitURL); err != nil {
		return
	}

	args := []string{"get", "-u", "-v", downloadURL}
	if !update {
		args = append(args[:1], args[2:]...)
	}

	goget := exec.Command("go", args...)
	goget.Stdin = os.Stdin
	goget.Stdout = os.Stdout

	errBuf := bytes.NewBuffer(nil)
	goget.Stderr = errBuf
	if err = goget.Run(); err == nil {
		return
	}

	if strings.Index(errBuf.String(), "no Go files in") > -1 {
		err = nil
		return
	}

	err = errors.Error(errBuf.String())
	return
}

func goBuild(gitURL, filename string) error {
	homeDir := os.Getenv("HOME")
	goDir := path.Join(homeDir, "go", "src", gitURL)

	gobuild := exec.Command("go", "build", "--buildmode", "plugin", "-o", filename, goDir)
	gobuild.Stdin = os.Stdin
	gobuild.Stdout = os.Stdout
	gobuild.Stderr = os.Stderr
	return gobuild.Run()
}

func getGitDownloadURL(gitURL string) (downloadURL string, err error) {
	var u *url.URL
	if u, err = url.Parse("http://" + gitURL); err != nil {
		return
	}

	spl := strings.Split(u.Path, "/")
	if len(spl) > 2 {
		spl = spl[:3]
	}

	// Update path
	u.Path = path.Join(spl...)

	// Set download URL
	downloadURL = u.String()[7:]
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
