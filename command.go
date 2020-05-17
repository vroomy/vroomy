package main

import (
	"fmt"
	"os"

	"github.com/hatchify/closer"
	parg "github.com/hatchify/parg"
)

func commandFromArgs() (cmd *parg.Command, err error) {
	var p *parg.Parg
	p = parg.New()

	p.AddHandler("", runService, "Runs vroomy server.\n  Accepts flags specified in config.toml.\n  Use `vroomy` or `vroomy -<flag>`")

	p.AddHandler("help", help, "Prints available commands and flags.\n  Use `vroomy help <command>` or `vroomy help <-flag>` to get more specific info.")
	p.AddHandler("test", test, "Tests the currently checked out version of plugin(s).\n  Accepts filtered trailing args to target specific plugins.\n  Use `vpm test` for all plugins, or `vpm test <plugin> <plugin>`")

	p.AddGlobalFlag(parg.Flag{
		Name:        "initialize",
		Help:        "Initializes only the specified plugins.\n  Allows optimized custom commands.\n  Use `vroomy -init <plugin> <plugin>`",
		Identifiers: []string{"-init", "-initialize"},
		Type:        parg.STRINGS,
	})

	for _, c := range cfg.CommandEntries {
		if _, ok := p.GetAllowedCommands()[c.Name]; ok {
			err = fmt.Errorf("error: duplicate command with name: %s", c.Name)
			return
		}

		p.AddHandler(c.Name, dynamicHandler{handler: c.Handler}.handleDynamicCmd, c.Usage+"\n  (Dynamically handled by "+c.Handler+")")
	}

	for _, f := range cfg.FlagEntries {
		usage := f.Usage
		if len(f.DefaultValue) != 0 {
			usage += "\n  Default: " + f.DefaultValue
		}

		p.AddGlobalFlag(parg.Flag{
			Name:        f.Name,
			Help:        usage,
			Identifiers: []string{"-" + f.Name},
			Value:       f.DefaultValue,
		})
	}

	cmd, err = parg.Validate()
	return
}

func runService(cmd *parg.Command) (err error) {
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

	var serviceName = cfg.Name
	if serviceName == "" {
		serviceName = "service"
	}

	out.Notification("Close request received. One moment please...")
	if err = svc.Close(); err != nil {
		err = fmt.Errorf("error encountered while closing %s: %v", serviceName, err)
		handleError(err)
	}

	out.Successf("Successfully closed %s!", serviceName)
	os.Exit(0)
	return
}

func help(cmd *parg.Command) (err error) {
	var serviceName = cfg.Name
	if serviceName == "" {
		serviceName = "Vroomy"
	}

	var prefix = "Usage ::\n\n# " + serviceName + "\n"

	if cmd == nil {
		out.Notification(prefix + parg.Help(true))
		return
	}

	out.Notification(prefix + cmd.Help(true))
	return
}

func test(cmd *parg.Command) (err error) {
	out.Notificationf("Testing plugin compatibility...")

	var serviceName = cfg.Name
	if serviceName == "" {
		serviceName = "service"
	}

	if err = initService(); err != nil {
		out.Error("Init test failed :(")

		err = fmt.Errorf("error encountered while initializing %s: %v", serviceName, err)
		handleError(err)
	}

	out.Notification("Initialized plugins successfully!")
	out.Notification("Closing...")

	if err = svc.Close(); err != nil {
		out.Error("Close test failed :(")

		err = fmt.Errorf("error encountered while closing %s: %v", serviceName, err)
		handleError(err)
	}

	out.Notification("Closed plugins successfully!")
	out.Success("Test complete!")
	return
}
