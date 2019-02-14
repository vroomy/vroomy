package service

import (
	"fmt"
	"plugin"

	"github.com/BurntSushi/toml"
	"github.com/Hatch1fy/errors"
	"github.com/Hatch1fy/httpserve"
	"github.com/PathDNA/atoms"
)

const (
	// ErrInvalidTLSDirectory is returned when a tls directory is unset when the tls port has been set
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
)

// New will return a new instance of service
func New(cfgname string) (sp *Service, err error) {
	var s Service
	if _, err = toml.DecodeFile(cfgname, &s.cfg); err != nil {
		return
	}

	s.srv = httpserve.New()

	if err = s.initPlugins(); err != nil {
		return
	}

	if err = s.initRoutes(); err != nil {
		return
	}

	sp = &s
	return
}

// Service manages the web service
type Service struct {
	cfg Config
	srv *httpserve.Serve
	p   plugins
	// Closed state
	closed atoms.Bool
}

func (s *Service) initPlugins() (err error) {
	s.p = make(plugins)

	fmt.Println("Initing plugins", s.cfg.Plugins)
	if len(s.cfg.Plugins) == 0 {
		return
	}

	for _, filename := range s.cfg.Plugins {
		var key string
		if key, err = getPluginKey(filename); err != nil {
			return
		}

		fmt.Println("Initing plugin", filename, key)

		if s.p[key], err = plugin.Open(filename); err != nil {
			return
		}
		fmt.Println("Plugin opened", key)
	}

	return
}

func (s *Service) initRoutes() (err error) {
	fmt.Println("Initing routes!")

	for i, r := range s.cfg.Routes {
		fmt.Println("Initing route!", r)
		if err = r.init(s.p); err != nil {
			return fmt.Errorf("error initializing route #%d (%v): %v", i, r, err)
		}

		fmt.Printf("Listening to: %v\n", r.String())
		s.srv.GET(r.HTTPPath, r.handler)
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
