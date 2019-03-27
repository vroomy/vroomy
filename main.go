package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Hatch1fy/vroomie/service"
	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/toolkit/closer"

	_ "github.com/lib/pq"
)

func main() {
	var (
		out    *journaler.Journaler
		svc    *service.Service
		config string
		update bool
		err    error
	)

	flag.StringVar(&config, "config", "./config.toml", "Location of configuration file")
	flag.BoolVar(&update, "update", false, "Whether or not to update all plugins on start-up")
	flag.Parse()

	out = journaler.New("Vroomie")
	out.Notification("Hello there! One moment, initializing..")
	out.Notification("Configuration location: %s", config)
	if update {
		out.Notification("Plugin update flag enabled")
	}

	out.Notification("Starting service")
	if svc, err = service.New(config, update); err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	closer := closer.New()
	go func() {
		if err := svc.Listen(); err != nil {
			closer.Close(err)
		}
	}()

	go func() {
		time.Sleep(time.Millisecond * 300)
		var msg string
		port := svc.Port()
		tlsPort := svc.TLSPort()
		if port > 0 && tlsPort > 0 {
			msg = fmt.Sprintf("ports %d (HTTP) and %d (HTTPS)", port, tlsPort)
		} else if port > 0 {
			msg = fmt.Sprintf("port %d (HTTP)", port)
		} else {
			msg = fmt.Sprintf("port %d (HTTPS)", tlsPort)
		}

		out.Success("HTTP is now listening on %s", msg)
	}()

	if err = closer.Wait(); err != nil {
		log.Fatal(err)
	}

	out.Notification("*Catch*")
	out.Notification("Close request received, one moment..")

	if err = svc.Close(); err != nil {
		log.Fatal(err)
	}

	out.Success("Service has been closed")
	os.Exit(0)
}
