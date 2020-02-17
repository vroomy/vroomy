package service

import (
	"fmt"
	"os"
	"plugin"
	"strings"

	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/httpserve"
	"github.com/hatchify/atoms"
	"github.com/vroomy/plugins"
)

const (
	// ErrInvalidTLSDirectory is returned when a tls directory is unset when the tls port has been set
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
	// ErrInvalidInitializationFunc is returned when an unsupported initialization function is encountered
	ErrInvalidInitializationFunc = errors.Error("unsupported initialization func encountered")
	// ErrProtectedFlag is returned when a protected flag is used
	ErrProtectedFlag = errors.Error("cannot use protected flag")
)

// New will return a new instance of service
func New(cfg *Config) (sp *Service, err error) {
	var s Service
	s.cfg = cfg

	if err = os.Chdir(s.cfg.Dir); err != nil {
		err = fmt.Errorf("error changing directory: %v", err)
		return
	}

	if s.plog, err = newPanicLog(); err != nil {
		return
	}

	if err = initDir("data"); err != nil {
		err = fmt.Errorf("error initializing data directory: %v", err)
		return
	}

	if err = initDir("plugins"); err != nil {
		err = fmt.Errorf("error changing plugins directory: %v", err)
		return
	}

	s.srv = httpserve.New()
	if err = s.loadPlugins(); err != nil {
		err = fmt.Errorf("error loading plugins: %v", err)
		return
	}

	if err = s.initPlugins(); err != nil {
		err = fmt.Errorf("error initializing plugins: %v", err)
		return
	}

	if err = s.initGroups(); err != nil {
		err = fmt.Errorf("error initializing groups: %v", err)
		return
	}

	if err = s.initRoutes(); err != nil {
		err = fmt.Errorf("error initializing routes: %v", err)
		return
	}

	sp = &s
	return
}

// Service manages the web service
type Service struct {
	cfg *Config
	srv *httpserve.Serve
	p   *plugins.Plugins

	plog *panicLog
	// Closed state
	closed atoms.Bool
}

func (s *Service) loadPlugins() (err error) {
	if s.p, err = plugins.New("plugins"); err != nil {
		err = fmt.Errorf("error initializing plugins manager: %v", err)
		return
	}

	if len(s.cfg.Plugins) == 0 {
		return
	}

	for _, pluginKey := range s.cfg.Plugins {
		var key string
		if key, err = s.p.New(pluginKey, s.cfg.PerformUpdate); err != nil {
			err = fmt.Errorf("error creating new plugin for key \"%s\": %v", pluginKey, err)
			return
		}

		s.cfg.pluginKeys = append(s.cfg.pluginKeys, key)
	}

	if err = s.p.Initialize(); err != nil {
		err = fmt.Errorf("erorr initializing plugins: %v", err)
		return
	}

	return
}

func (s *Service) initGroups() (err error) {
	if len(s.cfg.Groups) == 0 {
		return
	}

	for _, group := range s.cfg.Groups {
		if err = group.init(s.p); err != nil {
			return
		}

		var (
			match *Group
			grp   httpserve.Group = s.srv
		)

		if match, err = s.cfg.getGroup(group.Group); err != nil {
			return
		} else if match != nil {
			grp = match.g
		}

		group.g = grp.Group(group.HTTPPath, group.handlers...)
	}

	return
}

func (s *Service) initRoutes() (err error) {
	// Set panic func
	s.srv.SetPanic(s.plog.Write)

	for i, r := range s.cfg.Routes {
		if err = r.init(s.p); err != nil {
			return fmt.Errorf("error initializing route #%d (%v): %v", i, r, err)
		}

		var (
			match *Group
			grp   httpserve.Group = s.srv
		)

		if match, err = s.cfg.getGroup(r.Group); err != nil {
			return
		} else if match != nil {
			grp = match.g
		}

		var fn func(string, ...httpserve.Handler)
		switch strings.ToLower(r.Method) {
		case "put":
			fn = grp.PUT
		case "post":
			fn = grp.POST
		case "delete":
			fn = grp.DELETE
		case "options":
			fn = grp.OPTIONS

		default:
			// Default case is GET
			fn = grp.GET
		}

		fn(r.HTTPPath, r.handlers...)
	}

	return
}

func (s *Service) initPlugins() (err error) {
	for _, pluginKey := range s.cfg.pluginKeys {
		if err = s.initPlugin(pluginKey); err != nil {
			err = fmt.Errorf("error initializing %s: %v", pluginKey, err)
			return
		}
	}

	return
}

func (s *Service) initPlugin(pluginKey string) (err error) {
	var p *plugin.Plugin
	if p, err = s.p.Get(pluginKey); err != nil {
		return
	}

	var sym plugin.Symbol
	if sym, err = p.Lookup("OnInit"); err != nil {
		err = nil
		return
	}

	switch fn := sym.(type) {
	case func(p *plugins.Plugins, env map[string]string) error:
		return fn(s.p, s.cfg.Environment)

	case func(p *plugins.Plugins, flags, env map[string]string) error:
		return fn(s.p, s.cfg.Flags, s.cfg.Environment)

	default:
		return ErrInvalidInitializationFunc
	}
}

func (s *Service) getHTTPListener() (l listener) {
	if s.cfg.TLSPort > 0 {
		// TLS port exists, return a new upgrader pointing to the configured tls port
		return httpserve.NewUpgrader(s.cfg.TLSPort)
	}

	// TLS port does not exist, return the raw httpserve.Serve
	return s.srv
}

func (s *Service) listenHTTP(errC chan error) {
	if s.cfg.Port == 0 {
		// HTTP port not set, return
		return
	}

	// Get http listener
	// Note: If TLS is set, an httpserve.Upgrader will be returned
	l := s.getHTTPListener()

	// Attempt to listen to HTTP with the configured port
	errC <- l.Listen(s.cfg.Port)
}

func (s *Service) listenHTTPS(errC chan error) {
	if s.cfg.TLSPort == 0 {
		// HTTPS port not set, return
		return
	}

	if len(s.cfg.TLSDir) == 0 {
		// Cannot serve TLS without a tls directory, send error down channel and return
		errC <- ErrInvalidTLSDirectory
		return
	}

	// Attempt to listen to HTTPS with the configured tls port and directory
	errC <- s.srv.ListenTLS(s.cfg.TLSPort, s.cfg.TLSDir)
}

// Listen will listen to the configured port
func (s *Service) Listen() (err error) {
	// Initialize error channel
	errC := make(chan error, 2)
	// Listen to HTTP (if needed)
	go s.listenHTTP(errC)
	// Listen to HTTPS (if needed)
	go s.listenHTTPS(errC)
	// Return any error which may come down the error channel
	return <-errC
}

// Port will return the current HTTP port
func (s *Service) Port() uint16 {
	return s.cfg.Port
}

// TLSPort will return the current HTTPS port
func (s *Service) TLSPort() uint16 {
	return s.cfg.TLSPort
}

// Close will close the selected service
func (s *Service) Close() (err error) {
	if !s.closed.Set(true) {
		return errors.ErrIsClosed
	}

	var errs errors.ErrorList
	errs.Push(s.p.Close())
	errs.Push(s.plog.Close())
	return errs.Err()
}
