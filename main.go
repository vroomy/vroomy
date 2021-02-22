package main

import (
	"github.com/gdbu/scribe"
	"github.com/hatchify/closer"
	"github.com/vroomy/config"
)

// DefaultConfigLocation is the default configuration location
const DefaultConfigLocation = "./config.toml"

var (
	svc *Service
	cfg *config.Config

	clsr *closer.Closer

	out *scribe.Scribe
)

func main() {
	// Get runtime commmand
	cmd := setupRuntime()

	// Run specified action
	if err := cmd.Exec(); err != nil {
		handleError(err)
	}
}
