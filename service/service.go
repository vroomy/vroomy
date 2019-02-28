package service

import (
	"fmt"
	"os"
	"plugin"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/httpserve"
	"github.com/Hatch1fy/vroomie/plugins"
	"github.com/PathDNA/atoms"
)

const (
	// ErrInvalidTLSDirectory is returned when a tls directory is unset when the tls port has been set
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
)

// New will return a new instance of service
func New(cfgname string, update bool) (sp *Service, err error) {
	var s Service
	if _, err = toml.DecodeFile(cfgname, &s.cfg); err != nil {
		return
	}

	if s.cfg.Dir == "" {
		s.cfg.Dir = "./"
	}

	if err = os.Chdir(s.cfg.Dir); err != nil {
		return
	}

	if err = initDir("data"); err != nil {
		return
	}

	if err = initDir("plugins"); err != nil {
		return
	}

	s.srv = httpserve.New()
	if err = s.initPlugins(); err != nil {
		return
	}

	if err = s.initGroups(); err != nil {
		return
	}

	if err = s.initRoutes(); err != nil {
		return
	}

	if err = s.onInitialization(); err != nil {
		return
	}

	sp = &s
	return
}

// Service manages the web service
type Service struct {
	cfg Config
	srv *httpserve.Serve
	p   *plugins.Plugins
	// Closed state
	closed atoms.Bool
}

func (s *Service) initPlugins() (err error) {
	if s.p, err = plugins.New("plugins"); err != nil {
		return
	}

	if len(s.cfg.Plugins) == 0 {
		return
	}

	for _, pluginKey := range s.cfg.Plugins {
		if _, err = s.p.New(pluginKey); err != nil {
			return
		}
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

		default:
			// Default case is GET
			fn = grp.GET
		}

		fn(r.HTTPPath, r.handlers...)
	}

	return
}

func (s *Service) onInitialization() (err error) {
	for _, onInitKey := range s.cfg.OnInit {
		var (
			key   string
			fnKey string
			//			args []string
		)

		if key, fnKey, _, err = getHandlerParts(onInitKey); err != nil {
			return
		}

		var p *plugin.Plugin
		if p, err = s.p.Get(key); err != nil {
			return
		}

		var sym plugin.Symbol
		if sym, err = p.Lookup(fnKey); err != nil {
			return
		}

		fn, ok := sym.(func(p *plugins.Plugins) error)
		if !ok {
			return fmt.Errorf("invalid init function, received %v", sym)
		}

		if err = fn(s.p); err != nil {
			return
		}
	}

	return
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

	return
}

type listener interface {
	Listen(port uint16) error
}
