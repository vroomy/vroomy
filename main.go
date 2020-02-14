package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hatchify/scribe"
	"github.com/missionMeteora/toolkit/closer"
	"github.com/vroomy/vroomy/service"
)

// DefaultConfigLocation is the default configuration location
const DefaultConfigLocation = "./config.toml"

var (
	out  *scribe.Scribe
	svc  *service.Service
	clsr *closer.Closer
)

func main() {
	var err error
	out = scribe.New("Vroomy")
	out.Notification("Hello there! One moment, initializing..")

	if err = initService(); err != nil {
		handleError(err)
	}
	defer svc.Close()

	clsr = closer.New()
	go listen()
	go notifyOfListening()

	if err = clsr.Wait(); err != nil {
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

func initService() (err error) {
	configLocation := os.Getenv("VROOMY_CONFIG")
	if len(configLocation) == 0 {
		configLocation = DefaultConfigLocation
	}

	var cfg *service.Config
	if cfg, err = service.NewConfig(configLocation); err != nil {
		err = fmt.Errorf("error encountered while reading configuration: %v", err)
		return
	}

	out.Notification("Starting service")
	if svc, err = service.New(cfg); err != nil {
		err = fmt.Errorf("error encountered while initializing service: %v", err)
		return
	}

	return
}

func listen() {
	var err error
	if err = svc.Listen(); err == nil {
		return
	}

	err = fmt.Errorf("error encountered while attempting to listen to HTTP: %v", err)
	clsr.Close(err)
}

func notifyOfListening() {
	time.Sleep(time.Millisecond * 300)
	msg := getListeningMessage(svc.Port(), svc.TLSPort())
	out.Successf("HTTP is now listening on %s", msg)
}

func getListeningMessage(port, tlsPort uint16) (msg string) {
	switch {
	case port > 0 && tlsPort > 0:
		msg = fmt.Sprintf("ports %d (HTTP) and %d (HTTPS)", port, tlsPort)
	case port > 0:
		msg = fmt.Sprintf("port %d (HTTP)", port)
	case tlsPort > 0:
		msg = fmt.Sprintf("port %d (HTTPS)", tlsPort)
	}

	return
}

func handleError(err error) {
	out.Errorf("Fatal error encountered: %v", err)
	os.Exit(1)
}
