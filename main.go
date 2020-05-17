package main

import (
	"fmt"
	"os"
	"strings"

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

	out *scribe.Scribe
)

func main() {
	outW := scribe.NewStdout()
	outW.SetTypePrefix(scribe.TypeNotification, ":: vroomy :: ")
	out = scribe.NewWithWriter(outW, "")
	out.Notification("Hello there! :: One moment, please... ::")

	// Load config location
	configLocation := os.Getenv("VROOMY_CONFIG")
	if len(configLocation) == 0 {
		configLocation = DefaultConfigLocation
	}

	// Parse config
	var err error
	if cfg, err = service.NewConfig(configLocation); err != nil {
		err = fmt.Errorf("error encountered while reading configuration: %v", err)
		return
	}

	// Get commmand
	var cmd *flag.Command
	if cmd, err = commandFromArgs(); err != nil {
		help(cmd)
		handleError(err)
	}

	// Parse flags
	cfg.Flags = make(map[string]string, len(cmd.Flags))
	for name, f := range cmd.Flags {
		switch val := f.Value.(type) {
		case string:
			cfg.Flags[name] = val
		case []string:
			cfg.Flags[name] = strings.Join(val, " ")
		default:
			err = fmt.Errorf("error: %s flag expects non-nil string argument: got \"%v\"", name, val)
			handleError(err)
		}
	}

	// Run command handler
	if err = cmd.Exec(); err != nil {
		handleError(err)
	}
}
