package main

import (
	"fmt"
	"strings"

	parg "github.com/hatchify/parg"
	"github.com/vroomy/config"
)

type dynamicHandler struct {
	handler string
}

func (dh dynamicHandler) handleDynamicCmd(cmd *parg.Command) (err error) {
	if err = initService(); err != nil {
		err = fmt.Errorf("error encountered while initializing %s: %v", cfg.Name, err)
		handleError(err)
	}

	out.Notificationf("Handling dynamic command: %s", dh.handler)

	// Parse command handler
	comps := strings.Split(dh.handler, ".")
	if len(comps) != 2 {
		return fmt.Errorf("error: unable to parse plugin and handler format <plugin>.<handler> from: %s", dh.handler)
	}

	// Check plugin
	p, err := svc.Plugins.Get(comps[0])
	if err != nil {
		return
	}

	// Check handler method
	sym, err := p.Lookup(comps[1])
	if err != nil {
		return
	}

	// Confirm plugin has supported handler syntax
	var handlerErr error
	switch fn := sym.(type) {
	case func(flags, env map[string]string) error:
		handlerErr = fn(cfg.Flags, cfg.Environment)
	case func(env map[string]string) error:
		handlerErr = fn(cfg.Environment)
	case func(cfg *config.Config) error:
		handlerErr = fn(cfg)
	case func() error:
		handlerErr = fn()
	default:
		handlerErr = fmt.Errorf("error: no valid header for handler \"%s\" found in \"%s\"", comps[1], comps[0])
	}

	if handlerErr != nil {
		out.Errorf("%s encounted an error: %v", dh.handler, handlerErr)
	}

	out.Notification("Closing...")

	if err = svc.Close(); err != nil {
		err = fmt.Errorf("error encountered while closing %s: %v", cfg.Name, err)
		return
	}

	out.Notification("Closed plugins successfully!")

	if handlerErr != nil {
		return handlerErr
	}

	out.Successf("Executed %s!", cmd.Action)
	return
}
