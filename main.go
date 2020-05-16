package main

import (
	"fmt"
	"os"

	"github.com/hatchify/closer"
	flag "github.com/hatchify/parg"
	"github.com/hatchify/scribe"
	"github.com/vroomy/vroomy/service"
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

	configLocation := os.Getenv("VROOMY_CONFIG")
	if len(configLocation) == 0 {
		configLocation = DefaultConfigLocation
	}

	var err error
	if cfg, err = service.NewConfig(configLocation); err != nil {
		err = fmt.Errorf("error encountered while reading configuration: %v", err)
		return
	}

	var cmd *flag.Command
	if cmd, err = commandFromArgs(); err != nil {
		help(cmd)
		handleError(err)
	}

	if err = cmd.Exec(); err != nil {
		handleError(err)
	}
}
