package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	parg "github.com/hatchify/parg"
	"github.com/vroomy/config"
)

type dynamicHandler struct {
	prehook  string
	handler  string
	posthook string
}

func (dh *dynamicHandler) runHook(cmd string) (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return
	}

	comps := strings.Split(cmd, " ")
	var args []string
	if len(comps) > 1 {
		args = comps[1:]
	}

	hook := exec.Command(comps[0], args...)
	hook.Dir = pwd

	if err = hook.Run(); err != nil {
		return
	}

	return
}

func (dh *dynamicHandler) runPrehook() (err error) {
	out.Notificationf("Running prehook: %s", dh.prehook)
	return dh.runHook(dh.prehook)
}

func (dh *dynamicHandler) runPosthook() (err error) {
	out.Notificationf("Running posthook: %s", dh.posthook)
	return dh.runHook(dh.posthook)
}

func (dh *dynamicHandler) handle(cmd *parg.Command) (err error) {
	if len(dh.prehook) > 0 {
		if err = dh.runPrehook(); err != nil {
			err = fmt.Errorf("error: could not run prehoook `%s` for cmd %s: %+v", dh.prehook, cmd.Action, err)
			return
		}
	}

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

	if len(dh.posthook) > 0 {
		if err = dh.runPosthook(); err != nil {
			err = fmt.Errorf("error: could not run posthoook `%s` for cmd %s: %+v", dh.posthook, cmd.Action, err)
			return
		}
	}

	out.Successf("Executed %s!", cmd.Action)
	return
}
