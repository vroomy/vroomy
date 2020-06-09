package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	flag "github.com/hatchify/parg"
	"github.com/hatchify/scribe"
	"github.com/vroomy/config"
)

// Load the action and config environment
func setupRuntime() (cmd *flag.Command) {
	// Setup logging
	outW = scribe.NewStdout()
	outW.SetTypePrefix(scribe.TypeNotification, ":: vroomy :: ")
	out = scribe.NewWithWriter(outW, "")

	// Load config location
	configLocation := os.Getenv("VROOMY_CONFIG")
	if len(configLocation) == 0 {
		configLocation = DefaultConfigLocation
	}

	// Parse config
	var cfgErr error
	cfg, cfgErr = config.NewConfig(configLocation)

	// Load command (apply config if available)
	var err error
	cmd, err = commandFromArgs()
	if err != nil {
		showHelp(nil)
		handleError(err)
		return
	}

	switch cmd.Action {
	case "version", "upgrade":
		// Global actions
	default:
		if cfgErr != nil {
			out.Warning("Warning :: No config set.")

			if cmd.Action == "help" {
				// We can ignore config errors if we're asking for help. Use default
				cfg = &config.Config{Name: "vroomy service"}
			} else {
				// Config is required
				handleError(cfgErr)
			}
		}

		// Parse flags into config
		if err = parseConfigFlagsFrom(cmd); err != nil {
			showHelp(cmd)

			handleError(err)
		}
	}

	return
}

// Starts server
func initService() (err error) {
	out.Notificationf("Starting %s...", cfg.Name)

	if svc, err = New(cfg, "data"); err != nil {
		err = fmt.Errorf("error encountered while initializing service: %v", err)
		return
	}

	return
}

//
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

	// Set requires first, override if provided elsewhere
	for _, c := range cfg.CommandEntries {
		if c.Name == cmd.Action {
			// We're running this command
			if len(c.Require) != 0 {
				cfg.Flags["require"] = c.Require
			}

			break
		}
	}

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
			err = fmt.Errorf("error: %s flag received unexpected argument type: unable to parse \"%+v\"", f.Name, f.Type)
			return
		}

		if len(cfg.Flags[name]) == 0 {
			if cmd.Action != "help" {
				// Needs argument unless asking for usage
				err = fmt.Errorf("error: \"-%s\" flag expects %s: got \"%+v\"", f.Name, f.Type.Expects(), f.Value)
				return
			}
			return
		}
	}

	return
}

func initDir(loc string) (err error) {
	if err = os.Mkdir(loc, 0744); err == nil {
		return
	}

	if os.IsExist(err) {
		return nil
	}

	return
}

type listener interface {
	Listen(port uint16) error
}
