// Command hello serves the routes in config.example.toml using a local plugin.
package main

import (
	"context"
	"log"
	"os"

	"github.com/vroomy/vroomy"

	// Register the example's plugins before main runs.
	_ "github.com/vroomy/vroomy/examples/hello/plugins"
)

func main() {
	configPath := "config.example.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	var (
		svc *vroomy.Vroomy
		err error
	)

	if svc, err = vroomy.New(configPath); err != nil {
		log.Fatal(err)
	}

	if err = svc.ListenUntilSignal(context.Background()); err != nil {
		log.Fatal(err)
	}
}
