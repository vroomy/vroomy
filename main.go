package main

import (
	"fmt"
	"os"

	"github.com/hatchify/closer"
	flag "github.com/hatchify/parg"
	"github.com/hatchify/scribe"
	"github.com/vroomy/service"
)

// DefaultConfigLocation is the default configuration location
const DefaultConfigLocation = "./config.toml"

var (
	svc *service.Service
	cfg *service.Config

	clsr *closer.Closer

	out  *scribe.Scribe
	outW *scribe.Stdout
)

func main() {
	var err error

	outW = scribe.NewStdout()
	outW.SetTypePrefix(scribe.TypeNotification, ":: vroomy :: ")
	out = scribe.NewWithWriter(outW, "")

	// Load config location
	configLocation := os.Getenv("VROOMY_CONFIG")
	if len(configLocation) == 0 {
		configLocation = DefaultConfigLocation
	}

	// Get commmand
	var cmd *flag.Command
	if cmd, err = commandFromArgs(); err != nil {
		help(cmd)
		handleError(err)
	}

	switch cmd.Action {
	case "help", "version", "upgrade":
		// No config needed
		cfg = &service.Config{Name: "vroomy"}
	default:
		// Parse config
		if cfg, err = service.NewConfig(configLocation); err != nil {
			handleError(fmt.Errorf("error encountered while reading configuration: %v", err))
			return
		}

		// Parse flags into config
		if err = parseConfigFlagsFrom(cmd); err != nil {
			help(cmd)
			handleError(err)
		}
	}

	// Run command handler
	if err = cmd.Exec(); err != nil {
		handleError(err)
	}
}
