package main

import (
	"flag"
	"log"
	"os"

	"github.com/hatchify/vroomie/service"
	"github.com/missionMeteora/toolkit/closer"
)

func main() {
	var (
		svc    *service.Service
		config string
		err    error
	)

	flag.StringVar(&config, "config", "./config.toml", "Location of configuration file")
	flag.Parse()

	if svc, err = service.New(config); err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	closer := closer.New()
	go func() {
		if err := svc.Listen(); err != nil {
			closer.Close(err)
		}
	}()

	if err = closer.Wait(); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
