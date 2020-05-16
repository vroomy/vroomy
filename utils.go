package main

import (
	"fmt"
	"os"
	"time"

	"github.com/vroomy/service"
)

func initService() (err error) {
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
