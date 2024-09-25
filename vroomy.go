package vroomy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gdbu/atoms"
	"github.com/gdbu/scribe"
	"github.com/hatchify/errors"
	"github.com/vroomy/httpserve"
)

const (
	// ErrInvalidTLSDirectory is returned when a tls directory is unset when the tls port has been set
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
	// ErrInvalidPreInitFunc is returned when an unsupported pre initialization function is encountered
	ErrInvalidPreInitFunc = errors.Error("unsupported header for Init func encountered")
	// ErrInvalidLoadFunc is returned when an unsupported initialization function is encountered
	ErrInvalidLoadFunc = errors.Error("unsupported header for Load func encountered")
	// ErrNotAddressable is returned when a plugin is not addressable
	ErrNotAddressable = errors.Error("provided backend must be addressable")
	// ErrInvalidDir is returned when a directory is empty
	ErrInvalidDir = errors.Error("invalid directory, cannot be empty")
	// ErrPluginKeyExists is returned when a plugin cannot be added because it already exists
	ErrPluginKeyExists = errors.Error("plugin cannot be added, key already exists")
	// ErrPluginNotLoaded is returned when a plugin namespace is provided that has not been loaded
	ErrPluginNotLoaded = errors.Error("plugin with that key has not been loaded")
	// ErrExpectedEndParen is returned when an ending parenthesis is missing
	ErrExpectedEndParen = errors.Error("expected ending parenthesis")
	// ErrInvalidPluginHandler is returned when a plugin handler is not valid
	ErrInvalidPluginHandler = errors.Error("plugin handler not valid")
)

// New will return a new instance of service
func New(configLocation string) (sp *Vroomy, err error) {
	var cfg *Config
	if cfg, err = NewConfig(configLocation); err != nil {
		return
	}

	if _, ok := cfg.Environment["dataDir"]; !ok {
		// Default if not set elsewhere
		cfg.Environment["dataDir"] = "data"
	}

	return NewWithConfig(cfg)
}

// NewWithConfig will return a new instance of service with a provided config
func NewWithConfig(cfg *Config) (vp *Vroomy, err error) {
	var v Vroomy
	v.cfg = cfg
	v.out = scribe.New("Vroomy")
	if err = os.Chdir(v.cfg.Dir); err != nil {
		err = fmt.Errorf("error changing directory: %v", err)
		return
	}

	if err = initDir(v.cfg.Environment["dataDir"]); err != nil {
		err = fmt.Errorf("error initializing data directory: %v", err)
		return
	}

	if err = initDir("build"); err != nil {
		err = fmt.Errorf("error initializing plugin build directory: %v", err)
		return
	}

	v.srv = httpserve.New()
	v.srv.SetOnError(v.cfg.ErrorLogger)
	v.pm = p.Loaded()

	if err = v.initPlugins(); err != nil {
		return
	}

	if err = v.loadPlugins(); err != nil {
		return
	}

	if err = v.initGroups(); err != nil {
		err = fmt.Errorf("error initializing groups: %v", err)
		return
	}

	if err = v.initRoutes(); err != nil {
		err = fmt.Errorf("error initializing routes: %v", err)
		return
	}

	vp = &v
	return
}

// Vroomy manages the web service
type Vroomy struct {
	cfg *Config
	srv *httpserve.Serve

	out *scribe.Scribe

	pm map[string]Plugin

	// Closed state
	closed atoms.Bool
}

func (v *Vroomy) initPlugins() (err error) {
	// Call Init(flags, env) for each initialized plugin
	for pluginKey, plugin := range v.pm {
		if err = plugin.Init(v.cfg.Environment); err != nil {
			err = fmt.Errorf("error loading plugin <%s>: %v", pluginKey, err)
			return
		}

		v.out.Successf("Initialized %s", pluginKey)
	}

	return
}

func (v *Vroomy) initGroups() (err error) {
	if len(v.cfg.Groups) == 0 {
		return
	}

	//filter, ok := v.cfg.Flags["require"]
	for _, group := range v.cfg.Groups {
		// TODO: Document what this does and uncomment
		//if ok {
		//	var hasPlugin = false
		//	for _, handler := range group.Handlers {
		//		if ok && strings.Contains(filter, strings.Split(handler, ".")[0]) {
		//			hasPlugin = true
		//			break
		//		}
		//	}
		//
		//	if !hasPlugin {
		//		continue
		//	}
		//}

		if err = v.initRouteGroup(group); err != nil {
			return
		}
	}

	return
}

func (v *Vroomy) initRouteGroup(g *RouteGroup) (err error) {
	for _, handlerKey := range g.Handlers {
		var h httpserve.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("initRouteGroup(): error getting handler for key of <%s>: %v", handlerKey, err)
			return
		}

		g.HTTPHandlers = append(g.HTTPHandlers, h)
	}

	var (
		match *RouteGroup
		grp   httpserve.Group = v.srv
	)

	if match, err = v.cfg.GetRouteGroup(g.Group); err != nil {
		return
	} else if match != nil {
		if grp = match.G; grp == nil {
			err = fmt.Errorf("parent group \"%s\" has not yet been initialized", match.Name)
			return
		}
	}

	g.G = grp.Group(g.HTTPPath, g.HTTPHandlers...)
	return
}

func (v *Vroomy) initRoutes() (err error) {
	// Set panic func
	v.srv.SetPanic(v.handlePanic)

	//filter, ok := v.cfg.Flags["require"]
	for _, r := range v.cfg.Routes {
		// TODO: Document what this does and uncomment
		//if ok {
		//	hasPlugin := true
		//	for _, handler := range r.Handlers {
		//		if ok && !strings.Contains(filter, strings.Split(handler, ".")[0]) {
		//			hasPlugin = false
		//			break
		//		}
		//	}
		//
		//	if !hasPlugin {
		//		continue
		//	}
		//}

		if err = v.initRoute(r); err != nil {
			return
		}

		var (
			match *RouteGroup
			grp   httpserve.Group = v.srv
		)

		if match, err = v.cfg.GetRouteGroup(r.Group); err != nil {
			return
		} else if match != nil {
			if match.G == nil {
				if err = v.initRouteGroup(match); err != nil {
					return
				}
			}

			grp = match.G
		}

		var fn func(string, ...httpserve.Handler) error
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

		if err = fn(r.HTTPPath, r.HTTPHandlers...); err != nil {
			return
		}
	}

	return
}

func (v *Vroomy) initRoute(r *Route) (err error) {
	for _, handlerKey := range r.Handlers {
		var h httpserve.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("initRoute(): error getting handler for key of <%s>: %v", handlerKey, err)
			return
		}

		r.HTTPHandlers = append(r.HTTPHandlers, h)
	}

	// Note: We are going to support the ability to serve a target even if we already
	// have handlers specified. This will allow us to add MW logic to file serving routes
	// in addition to our standard plugin routes.
	// TODO: Determine if we want to move file serving to a plugin approach, and remove it
	// from the core vroomy offerings
	if len(r.Target) == 0 {
		// No target is set, bail out now
		return
	}

	return
}

func (v *Vroomy) setDependencies(pluginKey string, dm dependencyMap) (err error) {
	var pi Plugin
	if pi, err = v.getPlugin(pluginKey); err != nil {
		return
	}

	rval := reflect.ValueOf(pi)
	if rval.Kind() == reflect.Ptr {
		rval = rval.Elem()
	}

	for depKey, indices := range dm {
		field := getField(rval, indices)
		if err = v.setBackend(field, depKey); err != nil {
			return
		}
	}

	return pi.Load(v.cfg.Environment)
}

func (v *Vroomy) getPlugin(key string) (pi Plugin, err error) {
	var ok bool
	if pi, ok = v.pm[key]; !ok {
		err = fmt.Errorf("plugin with key of <%s> has not been registered", key)
		return
	}

	return
}

func (v *Vroomy) getReference(key string) (reference interface{}, err error) {
	var pi Plugin
	if pi, err = v.getPlugin(key); err != nil {
		return
	}

	if reference = pi.Backend(); reference == nil {
		// The provided value isn't an exact match, nor does it match the provided interface
		err = fmt.Errorf("cannot call backend for plugin <%s>, provided value is nil", key)
		return
	}

	return
}

func (v *Vroomy) setBackend(backend reflect.Value, key string) (err error) {
	elem := backend.Elem()
	if !elem.CanSet() {
		return ErrNotAddressable
	}

	var reference interface{}
	if reference, err = v.getReference(key); err != nil {
		return
	}

	beVal := reflect.ValueOf(reference)
	if err = canSet(elem, beVal); err != nil {
		return
	}

	elem.Set(beVal)
	return
}

func (v *Vroomy) loadPlugins() (err error) {
	dms := makeDependenciesMap(v.pm)
	if err = dms.Validate(); err != nil {
		return
	}

	var count int
	if err = dms.Load(func(pluginKey string, dm dependencyMap) (err error) {
		if err = v.setDependencies(pluginKey, dm); err != nil {
			err = fmt.Errorf("error loading plugin <%s>: %v", pluginKey, err)
			return
		}

		count++
		v.out.Successf("Loaded %s (%d/%d)", pluginKey, count, len(dms))
		return
	}); err != nil {
		return
	}

	// Call Init(flags, env) for each initialized plugin
	//for pluginKey, plugin := range pluginList {
	//	if err = plugin.Load(); err != nil {
	//		err = fmt.Errorf("error initializing plugin <%s>: %v", pluginKey, err)
	//		return
	//	}
	//
	//	v.out.Successf("Loaded %s", pluginKey)
	//}

	return
}

func (v *Vroomy) getHTTPListener() (l listener) {
	if v.cfg.TLSPort > 0 && !v.cfg.AllowNonTLS {
		// TLS port exists, return a new upgrader pointing to the configured tls port
		return httpserve.NewUpgrader(v.cfg.TLSPort)
	}

	// TLS port does not exist, return the raw httpserve.Serve
	return v.srv
}

func (v *Vroomy) listenHTTP(errC chan error) {
	if v.cfg.Port == 0 {
		// HTTP port not set, return
		return
	}

	// Get http listener
	// Note: If TLS is set, an httpserve.Upgrader will be returned
	l := v.getHTTPListener()

	// Attempt to listen to HTTP with the configured port
	errC <- l.Listen(v.cfg.Port)
}

func (v *Vroomy) listenHTTPS(errC chan error) {
	if v.cfg.TLSPort == 0 {
		// HTTPS port not set, return
		return
	}

	switch {
	case v.cfg.hasTLSDir():
		// Attempt to listen to HTTPS with the configured tls port and directory
		errC <- v.srv.ListenTLS(v.cfg.TLSPort, v.cfg.TLSDir)
	case v.cfg.hasAutoCert():
		ac, err := v.cfg.autoCertConfig()
		if err != nil {
			errC <- err
		}

		// Attempt to listen to HTTPS with the configured tls port and directory
		errC <- v.srv.ListenAutoCertTLS(v.cfg.TLSPort, ac)
	default:
		// Cannot serve TLS without a tls directory, send error down channel and return
		errC <- ErrInvalidTLSDirectory
		return

	}
}

func (v *Vroomy) handlePanic(in interface{}) {
	v.out.Errorf("Panic caught:\n%v\n%s\n\n", in, string(debug.Stack()))
}

// Listen will listen to the configured port
func (v *Vroomy) Listen(ctx context.Context) (err error) {
	// Initialize error channel
	errC := make(chan error, 2)
	// Listen to HTTP (if needed)
	go v.listenHTTP(errC)
	// Listen to HTTPS (if needed)
	go v.listenHTTPS(errC)

	timer := time.NewTimer(time.Millisecond * 100)

	// Wait for one of the following:
	// - Timer to end, which means we are listening succesfully
	// - Error to come down error channel, which means an error occurred during listening
	// - Context is finished, which means the caller no longer needing this action to continue
	select {
	case <-timer.C:
		v.listenNotification()
	case err = <-errC:
		return
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for one of the following:
	// - Error to come down error channel, which means an error occurred during listening
	// - Context is finished, which means the caller no longer needing this action to continue
	select {
	case err = <-errC:
		return
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Listen will listen to the configured port
func (v *Vroomy) ListenUntilSignal(ctx context.Context) (err error) {
	vctx, cancel := context.WithCancel(ctx)
	go v.onClose(cancel)

	if err = v.Listen(vctx); err == context.Canceled {
		err = nil
	}

	var errs errors.ErrorList
	errs.Push(err)
	errs.Push(v.Close())
	return errs.Err()
}

// Port will return the current HTTP port
func (v *Vroomy) Port() uint16 {
	return v.cfg.Port
}

// TLSPort will return the current HTTPS port
func (v *Vroomy) TLSPort() uint16 {
	return v.cfg.TLSPort
}

// Close will close the selected service
func (v *Vroomy) Close() (err error) {
	if !v.closed.Set(true) {
		return errors.ErrIsClosed
	}

	var errs errors.ErrorList
	errs.Push(v.srv.Close())
	for key, p := range v.pm {
		if err := p.Close(); err != nil {
			err = fmt.Errorf("error closing <%s>: %v", key, err)
			errs.Push(err)
		}
	}
	return errs.Err()
}

func (v *Vroomy) listenNotification() {
	var msg string
	switch {
	case v.TLSPort() > 0 && v.Port() > 0:
		msg = fmt.Sprintf("Listening on ports %d (HTTPS) and %d (HTTP)", v.TLSPort(), v.Port())
	case v.TLSPort() > 0:
		msg = fmt.Sprintf("Listening on port %d (HTTPS)", v.TLSPort())
	case v.Port() > 0:
		msg = fmt.Sprintf("Listening on port %d (HTTP)", v.Port())
	}

	v.out.Success(msg)
}

// listenForClose will listen for closing signals (interrupt, terminate, abort, quit) and call close
func (v *Vroomy) onClose(fn func()) {
	// sc represents the signal channel
	sc := make(chan os.Signal, 1)
	// Listen for signal notifications
	// Discussion topic: Should we include SIGQUIT? If we catch the signal, we won't get to see the unwind
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	// Signal received
	<-sc
	fn()
}

// Register will register a plugin with a given key
func Register(key string, pi Plugin) error {
	return p.Register(key, pi)
}
