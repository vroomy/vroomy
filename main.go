package main

import (
	"fmt"
	"os"
	"time"

	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/toolkit/closer"
	"github.com/vroomy/server/service"

	_ "github.com/lib/pq"
)

var out *journaler.Journaler

func main() {
	var (
		cfg *service.Config
		svc *service.Service
		err error
	)

	out = journaler.New("Vroomie")
	out.Notification("Hello there! One moment, initializing..")
	if cfg, err = service.NewConfig("./config.toml"); err != nil {
		err = fmt.Errorf("error encountered while reading configuration: %v", err)
		handleError(err)
	}

	out.Notification("Starting service")
	if svc, err = service.New(cfg); err != nil {
		err = fmt.Errorf("error encountered while initializing service: %v", err)
		handleError(err)
	}
	defer svc.Close()

	closer := closer.New()
	go func() {
		if err := svc.Listen(); err != nil {
			err = fmt.Errorf("error encountered while attempting to listen to HTTP: %v", err)
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
		handleError(err)
	}

	out.Notification("*Catch*")
	out.Notification("Close request received, one moment..")

	if err = svc.Close(); err != nil {
		err = fmt.Errorf("error encountered while closing service: %v", err)
		handleError(err)
	}

	out.Success("Service has been closed")
	os.Exit(0)
}

func handleError(err error) {
	out.Error("Fatal error encountered: %v", err)
	os.Exit(1)
}
