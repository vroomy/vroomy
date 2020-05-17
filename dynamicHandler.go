package main

import (
	"fmt"
	"strings"

	parg "github.com/hatchify/parg"
	"github.com/vroomy/service"
)

type dynamicHandler struct {
	handler string
}

func (dh dynamicHandler) handleDynamicCmd(cmd *parg.Command) (err error) {
	var serviceName = cfg.Name
	if serviceName == "" {
		serviceName = "service"
	}

	if err = initService(); err != nil {
		err = fmt.Errorf("error encountered while initializing %s: %v", serviceName, err)
		handleError(err)
	}

	out.Notification("Initialized plugins successfully!")

	// Parse command handler
	out.Notificationf("Handling: %s", dh.handler)

	comps := strings.Split(dh.handler, ".")
	if len(comps) != 2 {
		return fmt.Errorf("error: unable to parse plugin and handler format <plugin>.<handler> from: %s", dh.handler)
	}
	p, err := svc.Plugins.Get(comps[0])
	if err != nil {
		return
	}

	sym, err := p.Lookup(comps[1])
	if err != nil {
		return
	}

	var handlerErr error
	switch fn := sym.(type) {
	case func(flags, env map[string]string) error:
		handlerErr = fn(cfg.Flags, cfg.Environment)
	case func(env map[string]string) error:
		handlerErr = fn(cfg.Environment)
	case func(cfg *service.Config) error:
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
		out.Error("Close test failed :(")

		err = fmt.Errorf("error encountered while closing %s: %v", serviceName, err)
		handleError(err)
	}

	out.Notification("Closed plugins successfully!")

	if handlerErr == nil {
		out.Successf("Executed %s!", cmd.Action)
	} else {
		handleError(handlerErr)
	}

	return
}
