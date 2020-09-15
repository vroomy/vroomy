package main

import (
	"fmt"
	"path"

	"github.com/hatchify/closer"
	parg "github.com/hatchify/parg"
)

func commandFromArgs() (cmd *parg.Command, err error) {
	var p *parg.Parg
	p = parg.New()

	p.AddHandler("", runService, "Runs vroomy server.\n  Accepts flags specified in config.toml.\n  Use `vroomy` or `vroomy -<flag>`")
	p.AddHandler("test", test, "Tests the currently built plugins for compatibility.\n  Closes service upon successful execution.\n  Use `vroomy test`")

	p.AddHandler("help", showHelp, "Prints available commands and flags.\n  Use `vroomy help <command>` or `vroomy help <-flag>` to get more specific info.")
	p.AddHandler("version", printVersion, "Prints current version of vroomy installation.\n  Use `vroomy version`")
	p.AddHandler("upgrade", upgrade, "Upgrades vroomy installation itself.\n  Skips if version is up to date.\n  Use `vroomy upgrade` or `vroomy upgrade <branch>`")

	p.AddGlobalFlag(parg.Flag{
		Name:        "require",
		Help:        "Initializes only the specified \"required\" plugins.\n  Allows optimized custom commands.\n  Use `vroomy -r <plugin> <plugin>`",
		Identifiers: []string{"-require", "-r"},
		Type:        parg.STRINGS,
	})

	p.AddGlobalFlag(parg.Flag{
		Name:        "dataDir",
		Help:        "Initializes backend data in provided directory.\n  Overrides default dir as well as value set in config.\n  Ignored when executing tests.\n  Use `vroomy -d <path_to_directory>`",
		Identifiers: []string{"-dataDir", "-d"},
	})

	p.AddGlobalFlag(parg.Flag{
		Name:        "config",
		Help:        "Initializes with config at specified location.\n  Overrides default config.\n  Use `vroomy -c <path_to_config>`",
		Identifiers: []string{"-config", "-c"},
	})

	addDynamicActions(p)

	cmd, err = parg.Validate()
	return
}

func addDynamicActions(p *parg.Parg) (err error) {
	if cfg == nil {
		// No dynamic commands or flags
		return
	}

	if cfg.CommandEntries != nil {
		// Handle config commands
		var dynamic *dynamicHandler
		for _, c := range cfg.CommandEntries {
			if _, ok := p.GetAllowedCommands()[c.Name]; ok {
				err = fmt.Errorf("error: duplicate command with name: %s", c.Name)
				return
			}

			dynamic = &dynamicHandler{prehook: c.Prehook, handler: c.Handler, posthook: c.Posthook}
			p.AddHandler(c.Name, dynamic.handle, c.Usage+"\n  (Dynamically handled by "+c.Handler+")")
		}
	}

	if cfg.FlagEntries != nil {
		// Handle config flags
		for _, f := range cfg.FlagEntries {
			usage := f.Usage
			if len(f.DefaultValue) != 0 {
				usage += "\n  Default: " + f.DefaultValue
			}

			if _, ok := p.GetGlobalFlags()[f.Name]; ok {
				err = fmt.Errorf("error: duplicate flag with name: %s", f.Name)
				return
			}

			p.AddGlobalFlag(parg.Flag{
				Name:        f.Name,
				Help:        usage,
				Identifiers: []string{"-" + f.Name},
				Value:       f.DefaultValue,
			})
		}
	}

	return
}

func runService(cmd *parg.Command) (err error) {
	out.Notificationf("Hello there! :: Starting %s :: One moment, please... ::", cfg.Name)

	var dataDir = cmd.StringFrom("dataDir")
	if dataDir != "" {
		cfg.Environment["dataDir"] = dataDir
	}

	if err = initService(); err != nil {
		return
	}
	defer svc.Close()

	clsr = closer.New()
	go listen()
	go notifyOfListening()

	if err = clsr.Wait(); err != nil {
		return
	}

	out.Notification("Close request received. One moment please...")
	if err = svc.Close(); err != nil {
		err = fmt.Errorf("error encountered while closing %s: %v", cfg.Name, err)
		return
	}

	out.Successf("Successfully closed %s!", cfg.Name)
	return
}

func showHelp(cmd *parg.Command) (err error) {
	var serviceName string
	if cfg != nil {
		serviceName = cfg.Name
	}

	if serviceName == "" {
		serviceName = "vroomy"
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
	var serviceName = cfg.Name
	if serviceName == "" {
		serviceName = "service"
	}

	out.Notificationf("Hello there! :: Testing %s Compatibility :: One moment, please... ::", serviceName)

	// Override for tests
	cfg.Environment["dataDir"] = path.Join(cfg.Environment["testDir"], "testData")

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
