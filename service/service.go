package service

import (
	"fmt"

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
	// Closed state
	closed atoms.Bool
}

func (s *Service) initRoutes() (err error) {
	for i, r := range s.cfg.Routes {
		if err = r.init(); err != nil {
			return fmt.Errorf("error initializing route #%d (%v): %v", i, r, err)
		}

		fmt.Printf("Listening to: %v\n", r.String())
		s.srv.GET(r.HTTPPath, r.serveHTTP)
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
