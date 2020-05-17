package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	flag "github.com/hatchify/parg"
	"github.com/vroomy/service"
)

func initService() (err error) {
	if len(cfg.Name) > 0 {
		out.Notificationf("Starting %s...", cfg.Name)
	} else {
		out.Notification("Starting vroomy service...")
	}

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

// Convert command flags to flag entities, handle defaults
func parseConfigFlagsFrom(cmd *flag.Command) (err error) {
	cfg.Flags = map[string]string{}

	// Add default values
	cmdFlags := cmd.Flags
	for _, entry := range cfg.FlagEntries {
		if entry.DefaultValue != "" {
			if _, ok := cmd.Flags[entry.Name]; !ok {
				// Set default value
				cfg.Flags[entry.Name] = entry.DefaultValue
			}
		}
	}

	// Parse flags, override defaults
	for name, f := range cmdFlags {
		switch f.Type {
		case flag.DEFAULT:
			cfg.Flags[name] = cmd.StringFrom(f.Name)
		case flag.STRINGS:
			cfg.Flags[name] = strings.Join(cmd.StringsFrom(f.Name), " ")
		case flag.BOOL:
			if cmd.BoolFrom(f.Name) {
				cfg.Flags[name] = "true"
			} else {
				cfg.Flags[name] = "false"
			}
		case flag.INT:
			cfg.Flags[name] = strconv.Itoa(cmd.IntFrom(f.Name))

		default:
			// Needs argument unless asking for usage
			err = fmt.Errorf("error: %s flag received unexpected argument type: unable to parse \"%+v\"", name, f.Type)
			return
		}

		if len(cfg.Flags[name]) == 0 {
			if cmd.Action != "help" {
				// Needs argument unless asking for usage
				err = fmt.Errorf("error: %s flag expects %s: got \"%+v\"", name, f.Type.Expects(), f.Value)
				return
			}
			return
		}
	}

	return
}
