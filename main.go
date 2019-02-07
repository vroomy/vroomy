package main

import (
	"log"
	"os"

	"github.com/hatchify/vroomie/service"
	"github.com/missionMeteora/toolkit/closer"
)

func main() {
	var (
		svc *service.Service
		err error
	)

	if svc, err = service.New("./config.toml"); err != nil {
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
