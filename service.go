package main

import (
	"fmt"
	"os"
	"path"
	"plugin"
	"runtime/debug"
	"strings"

	"github.com/gdbu/atoms"
	"github.com/gdbu/scribe"
	"github.com/hatchify/errors"
	"github.com/vroomy/common"
	"github.com/vroomy/config"
	"github.com/vroomy/httpserve"
	"github.com/vroomy/plugins"
)

const (
	// ErrInvalidTLSDirectory is returned when a tls directory is unset when the tls port has been set
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
	// ErrInvalidPreInitFunc is returned when an unsupported pre initialization function is encountered
	ErrInvalidPreInitFunc = errors.Error("unsupported header for Init func encountered")
	// ErrInvalidLoadFunc is returned when an unsupported initialization function is encountered
	ErrInvalidLoadFunc = errors.Error("unsupported header for Load func encountered")
)

// New will return a new instance of service
func New(cfg *config.Config) (sp *Service, err error) {
	var s Service
	s.cfg = cfg
	s.out = scribe.New("Vroomy")
	if err = os.Chdir(s.cfg.Dir); err != nil {
		err = fmt.Errorf("error changing directory: %v", err)
		return
	}

	if err = initDir(s.cfg.Environment["dataDir"]); err != nil {
		err = fmt.Errorf("error initializing data directory: %v", err)
		return
	}

	if err = initDir("build"); err != nil {
		err = fmt.Errorf("error initializing plugin build directory: %v", err)
		return
	}

	s.srv = httpserve.New()
	if err = s.initPlugins(); err != nil {
		err = fmt.Errorf("error loading plugins: %v", err)
		return
	}

	if err = s.loadPlugins(); err != nil {
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

	// TODO: Move this to docs/testing only?
	if err = s.initRouteExamples(); err != nil {
		err = fmt.Errorf("error initializing routes: %v", err)
		return
	}

	sp = &s
	return
}

// Service manages the web service
type Service struct {
	cfg     *config.Config
	srv     *httpserve.Serve
	Plugins *plugins.Plugins

	out *scribe.Scribe

	// Closed state
	closed atoms.Bool
}

func pluginName(key string) (name string) {
	comps := strings.Split(key, " as ")
	if len(comps) > 1 {
		return comps[1]
	}

	key = strings.Split(key, "#")[0]
	key = strings.Split(key, "@")[0]
	_, key = path.Split(key)

	name = key
	return
}

func pluginAlias(key string) (name string) {
	comps := strings.Split(key, " as ")
	if len(comps) > 1 {
		return comps[1]
	}

	key = strings.Split(key, "#")[0]
	key = strings.Split(key, "@")[0]
	key = strings.Split(key, "-")[0]
	_, key = path.Split(key)

	name = key
	return
}

func (s *Service) initPlugins() (err error) {
	if s.Plugins, err = plugins.New("build"); err != nil {
		err = fmt.Errorf("error initializing plugins manager: %v", err)
		return
	}

	if len(s.cfg.Plugins) == 0 {
		return
	}

	filter, ok := s.cfg.Flags["require"]
	for _, pluginKey := range s.cfg.Plugins {
		if ok && !strings.Contains(filter, pluginAlias(pluginKey)) && !strings.Contains(filter, pluginName(pluginKey)) {
			continue
		}

		var key string
		if key, err = s.Plugins.New(pluginKey, s.cfg.PerformUpdate); err != nil {
			err = fmt.Errorf("error creating new plugin for key \"%s\": %v", pluginKey, err)
			return
		}

		s.cfg.PluginKeys = append(s.cfg.PluginKeys, key)
	}

	if err = s.Plugins.Initialize(); err != nil {
		err = fmt.Errorf("error initializing plugins: %v", err)
		return
	}

	// Call Init(flags, env) for each initialized plugin
	for _, pluginKey := range s.cfg.PluginKeys {
		// Run init first to set data/env/flags and external deps
		if err = s.initPlugin(pluginKey); err != nil {
			err = fmt.Errorf("error initializing %s: %v", pluginKey, err)
			return
		}

		out.Notificationf("Initialized %s", pluginKey)
	}

	return
}

func (s *Service) initGroups() (err error) {
	if len(s.cfg.Groups) == 0 {
		return
	}

	filter, ok := s.cfg.Flags["require"]
	for _, group := range s.cfg.Groups {
		if ok {
			var hasPlugin = false
			for _, handler := range group.Handlers {
				if ok && strings.Contains(filter, strings.Split(handler, ".")[0]) {
					hasPlugin = true
					break
				}
			}

			if !hasPlugin {
				continue
			}
		}

		if err = s.initGroup(group); err != nil {
			return
		}
	}

	return
}

func (s *Service) initGroup(group *config.Group) (err error) {
	if err = group.Init(s.Plugins); err != nil {
		return
	}

	var (
		match *config.Group
		grp   httpserve.Group = s.srv
	)

	if match, err = s.cfg.GetGroup(group.Group); err != nil {
		return
	} else if match != nil {
		grp = match.G
	}

	group.G = grp.Group(group.HTTPPath, group.HTTPHandlers...)
	return
}

func (s *Service) initRoutes() (err error) {
	// Set panic func
	s.srv.SetPanic(s.handlePanic)

	filter, ok := s.cfg.Flags["require"]
	for i, r := range s.cfg.Routes {
		if ok {
			var hasPlugin = true
			for _, handler := range r.Handlers {
				if ok && !strings.Contains(filter, strings.Split(handler, ".")[0]) {
					hasPlugin = false
					break
				}
			}

			if !hasPlugin {
				continue
			}
		}

		if err = r.Init(s.Plugins); err != nil {
			return fmt.Errorf("error initializing route #%d (%v): %v", i, r, err)
		}

		var (
			match *config.Group
			grp   httpserve.Group = s.srv
		)

		if match, err = s.cfg.GetGroup(r.Group); err != nil {
			return
		} else if match != nil {
			if match.G == nil {
				s.initGroup(match)
			}

			grp = match.G
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

		fn(r.HTTPPath, r.HTTPHandlers...)
	}

	return
}

func (s *Service) initRouteExamples() (err error) {
	s.cfg.ExampleResponses = make(map[string]*config.Response)
	var needsParentRes = []*config.Response{}
	for _, res := range s.cfg.Responses {
		s.cfg.ExampleResponses[res.Name] = res
		if len(strings.TrimSpace(res.Parent)) > 0 {
			needsParentRes = append(needsParentRes, res)
		}
	}

	for _, res := range needsParentRes {
		if _, ok := s.cfg.ExampleResponses[res.Parent]; ok {
			res.InheritFrom(s.cfg.ExampleResponses)
		} else {
			out.Warningf("Unable to find parent (%s) for response: %s", res.Parent, res.Name)
		}
	}

	s.cfg.ExampleRequests = make(map[string]*config.Request)
	var needsParentReq = []*config.Request{}
	for _, req := range s.cfg.Requests {
		s.cfg.ExampleRequests[req.Name] = req

		if len(req.Group) > 0 {
			var g *config.Group
			if g, err = s.cfg.GetGroup(req.Group); err != nil {
				out.Warningf("Unable to find group (%s) for request: ", req.Name)
			}

			if g.Requests == nil {
				g.Requests = make(map[string]*config.Request)
			}

			g.Requests[req.Name] = req
		}

		if len(strings.TrimSpace(req.Parent)) > 0 {
			needsParentReq = append(needsParentReq, req)
		}

		for _, resName := range req.Responses {
			if res, ok := s.cfg.ExampleResponses[resName]; ok {
				req.ResponseExamples = append(req.ResponseExamples, res)
			} else {
				out.Warningf("Unable to find response (%s) for request: %s", resName, req.Name)
			}
		}
	}

	for _, req := range needsParentReq {
		if _, ok := s.cfg.ExampleRequests[req.Parent]; ok {
			req.InheritFrom(s.cfg.ExampleRequests)
		} else {
			out.Warningf("Unable to find parent (%s) for request: %s", req.Parent, req.Name)
		}
	}

	return
}

func (s *Service) loadPlugins() (err error) {
	// Call Load(p common.Plugins) for each loaded plugin
	for _, pluginKey := range s.cfg.PluginKeys {
		// Run configure after all plugins init to set intra-service deps
		if err = s.loadPlugin(pluginKey); err != nil {
			err = fmt.Errorf("error loading %s: %v", pluginKey, err)
			return
		}

		out.Successf("Loaded %s", pluginKey)
	}

	return
}

func (s *Service) initPlugin(pluginKey string) (err error) {
	var p *plugin.Plugin
	if p, err = s.Plugins.Get(pluginKey); err != nil {
		return
	}

	var sym plugin.Symbol
	if sym, err = p.Lookup("Init"); err != nil {
		err = nil
		return
	}

	switch fn := sym.(type) {
	case func(flags, env map[string]string) error:
		return fn(s.cfg.Flags, s.cfg.Environment)
	case func(env map[string]string) error:
		return fn(s.cfg.Environment)
	case func() error:
		return fn()

	default:
		return ErrInvalidPreInitFunc

	}
}

func (s *Service) loadPlugin(pluginKey string) (err error) {
	var p *plugin.Plugin
	if p, err = s.Plugins.Get(pluginKey); err != nil {
		return
	}

	var sym plugin.Symbol
	if sym, err = p.Lookup("Load"); err != nil {
		// Legacy plugin support
		if sym, err = p.Lookup("OnInit"); err != nil {
			err = nil
			return
		}

		// Legacy init functions
		switch fn := sym.(type) {
		case func(p common.Plugins, flags, env map[string]string) error:
			return fn(s.Plugins, s.cfg.Flags, s.cfg.Environment)
		case func(p common.Plugins, env map[string]string) error:
			return fn(s.Plugins, s.cfg.Environment)
		case func(p common.Plugins) error:
			return fn(s.Plugins)
		case func() error:
			return fn()

		default:
			return ErrInvalidLoadFunc
		}
	}

	switch fn := sym.(type) {
	case func(p common.Plugins) error:
		return fn(s.Plugins)
	case func() error:
		return fn()

	default:
		return ErrInvalidLoadFunc
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

func (s *Service) handlePanic(v interface{}) {
	s.out.Errorf("Panic caught:\n%v\n%s\n\n", v, string(debug.Stack()))
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
	errs.Push(s.Plugins.Close())
	return errs.Err()
}
