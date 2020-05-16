package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hatchify/closer"
	parg "github.com/hatchify/parg"
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

	out.Notification("*Catch*")
	out.Notification("Close request received, one moment..")

	if err = svc.Close(); err != nil {
		err = fmt.Errorf("error encountered while closing service: %v", err)
		handleError(err)
	}

	out.Success("Service has been closed")
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
	out.Notificationf("Documenting postman config...")

	output := postmanFromConfig()
	file, _ := json.MarshalIndent(output, "", " ")
	_ = ioutil.WriteFile("postman.json", file, 0644)

	out.Success("Generated postman.json collection successfully!")
	return
}

func test(cmd *parg.Command) (err error) {
	out.Notificationf("Testing plugin compatibility...")

	if err = initService(); err != nil {
		out.Error("Init test failed :(")

		handleError(err)
	}

	out.Notification("Initialized plugins successfully!")
	out.Notification("Closing...")

	if err = svc.Close(); err != nil {
		out.Error("Close test failed :(")

		err = fmt.Errorf("error encountered while closing service: %v", err)
		handleError(err)
	}

	out.Notification("Closed plugins successfully!")
	out.Success("Test complete!")
	return
}
