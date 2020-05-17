package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hatchify/closer"
	parg "github.com/hatchify/parg"
	"github.com/vroomy/vroomy/postman"
)

func commandFromArgs() (cmd *parg.Command, err error) {
	var p *parg.Parg
	p = parg.New()

	p.AddHandler("", startServer, "Runs vroomy server.\n  Accepts flags specified in config.toml.\n  Use `vroomy` or `vroomy -<flag>`")

	p.AddHandler("help", help, "Prints available commands and flags.\n  Use `vroomy help <command>` to get more specific info.")

	p.AddHandler("doc", doc, "Outputs docs for specified config.\n  May support multiple formats.\n  Use `vroomy doc` or `vpm doc -format postman`")
	p.AddHandler("test", test, "Tests the currently checked out version of plugin(s).\n  Accepts filtered trailing args to target specific plugins.\n  Use `vpm test` for all plugins, or `vpm test <plugin> <plugin>`")

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

func startServer(cmd *parg.Command) (err error) {
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
	if cmd == nil {
		out.Success(parg.Help())
		return
	}

	out.Success(cmd.Help())
	return
}

func doc(cmd *parg.Command) (err error) {
	out.Notificationf("Generating Docs...")

	var p *postman.Postman
	if p, err = postman.FromConfig(cfg); err != nil {
		handleError(err)
	}

	var filename string
	if len(cfg.Name) > 0 {
		filename = strings.Replace(cfg.Name, " ", "_", -1)
	} else {
		filename = "vroomy"
	}

	filename += "_postman_collection.json"
	p.WriteToFile(filename)

	out.Successf("Generated \"%s\" collection successfully!", filename)
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
