package vroomy

import (
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gdbu/atoms"
	"github.com/gdbu/errors"
	"github.com/vroomy/httpserve"
)

var constructionMu sync.Mutex

const (
	// ErrPluginInUse is returned when a service already owns the registry's plugins.
	ErrPluginInUse = errors.Error("registered plugins are already owned by an active service")

	// ErrInvalidTLSDirectory is returned by Listen when TLS is enabled without
	// TLSDir or both AutoCertDir and AutoCertHosts.
	ErrInvalidTLSDirectory = errors.Error("invalid tls directory, cannot be empty when tls port has been set")
	// ErrInvalidPreInitFunc is a legacy sentinel; lifecycle signatures are now
	// enforced by the Plugin interface, and the current code does not return it.
	ErrInvalidPreInitFunc = errors.Error("unsupported header for Init func encountered")
	// ErrInvalidLoadFunc is a legacy sentinel not returned by the current code.
	ErrInvalidLoadFunc = errors.Error("unsupported header for Load func encountered")
	// ErrNotAddressable is returned when a dependency destination cannot be set.
	ErrNotAddressable = errors.Error("provided backend must be addressable")
	// ErrInvalidDir is a legacy sentinel not returned by the current code.
	ErrInvalidDir = errors.Error("invalid directory, cannot be empty")
	// ErrPluginKeyExists identifies duplicate registration keys.
	ErrPluginKeyExists = errors.Error("plugin cannot be added, key already exists")
	// ErrPluginNotLoaded identifies an unregistered plugin key.
	ErrPluginNotLoaded = errors.Error("plugin with that key has not been loaded")
	// ErrExpectedEndParen is returned when an ending parenthesis is missing
	ErrExpectedEndParen = errors.Error("expected ending parenthesis")
	// ErrInvalidPluginHandler identifies nil handlers or invalid handler signatures.
	ErrInvalidPluginHandler = errors.Error("plugin handler not valid")
)

// New loads a configuration file, defaults a missing dataDir environment key to
// "data", and constructs the service with NewWithConfig. It does not start listeners.
func New(configLocation string) (sp *Vroomy, err error) {
	var cfg *Config
	if cfg, err = NewConfig(configLocation); err != nil {
		return sp, err
	}

	if _, ok := cfg.Environment["dataDir"]; !ok {
		// Default if not set elsewhere
		cfg.Environment["dataDir"] = "data"
	}

	return NewWithConfig(cfg)
}

// NewWithConfig changes the process working directory to cfg.Dir, exclusively
// acquires registered plugins, initializes them, and registers routes. It retains
// and mutates cfg without loading includes or supplying defaults. On failure it
// closes every plugin whose Init was attempted and restores the working directory.
// A second service using the registry returns ErrPluginInUse until Close completes.
func NewWithConfig(cfg *Config) (vp *Vroomy, err error) {
	constructionMu.Lock()
	defer constructionMu.Unlock()
	if cfg == nil {
		return nil, fmt.Errorf("configuration must be non-nil")
	}

	var v Vroomy
	v.cfg = cfg
	v.registry = p
	if v.pm, err = v.registry.acquire(); err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = stderrors.Join(err, v.closePlugins())
			v.registry.release()
		}
	}()
	for key, plugin := range v.pm {
		if err = validatePlugin(plugin); err != nil {
			return nil, fmt.Errorf("validate plugin %q: %w", key, err)
		}
	}

	if v.dependencies, err = makeDependenciesMap(v.pm); err != nil {
		return nil, err
	}

	if err = v.dependencies.Validate(); err != nil {
		return nil, err
	}

	if err = v.dependencies.Load(func(key string, _ dependencyMap) error {
		v.pluginOrder = append(v.pluginOrder, key)
		return nil
	}); err != nil {
		return nil, err
	}

	var previousDir string
	if previousDir, err = os.Getwd(); err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	if err = os.Chdir(cfg.Dir); err != nil {
		return nil, fmt.Errorf("change directory to %q: %w", cfg.Dir, err)
	}

	defer func() {
		if err != nil {
			// Restore after rollback, so plugin Close sees the same directory as Init.
			rollbackErr := v.closePlugins()
			v.initialized = nil
			if restoreErr := os.Chdir(previousDir); restoreErr != nil {
				err = stderrors.Join(err, fmt.Errorf("restore working directory %q: %w", previousDir, restoreErr))
			}

			err = stderrors.Join(err, rollbackErr)
		}
	}()
	if err = initDir(cfg.Environment["dataDir"]); err != nil {
		return nil, fmt.Errorf("initialize data directory %q: %w", cfg.Environment["dataDir"], err)
	}

	v.srv = httpserve.New()
	v.srv.SetOnError(cfg.ErrorLogger)
	if err = v.initPlugins(); err != nil {
		return nil, err
	}

	if err = v.loadPlugins(); err != nil {
		return nil, err
	}

	if err = v.initGroups(); err != nil {
		return nil, fmt.Errorf("initialize groups: %w", err)
	}

	if err = v.initRoutes(); err != nil {
		return nil, fmt.Errorf("initialize routes: %w", err)
	}

	return &v, nil
}

// Vroomy manages a web service with exclusive ownership of registered plugins.
// Construction mutates its config and the process working directory. Plugins
// supply their own request concurrency guarantees; Close reverses dependency order.
type Vroomy struct {
	cfg *Config
	srv *httpserve.Serve

	pm           map[string]Plugin
	registry     *Plugins
	dependencies dependenciesMap
	pluginOrder  []string
	initialized  []string

	// Closed state
	closed atoms.Bool
}

func (v *Vroomy) initPlugins() (err error) {
	// Call Init(env) for each registered plugin before dependency injection.
	for _, pluginKey := range v.pluginOrder {
		plugin := v.pm[pluginKey]
		// Include a failing Init: it may already own partially acquired resources.
		v.initialized = append(v.initialized, pluginKey)
		if err = plugin.Init(v.cfg.Environment); err != nil {
			err = fmt.Errorf("error loading plugin <%s>: %w", pluginKey, err)
			return err
		}

		log.Printf("Vroomy: Initialized %s\n", pluginKey)
	}

	return err
}

func (v *Vroomy) initGroups() (err error) {
	if len(v.cfg.Groups) == 0 {
		return err
	}

	for _, group := range v.cfg.Groups {
		if err = v.initRouteGroup(group); err != nil {
			return err
		}
	}

	return err
}

func (v *Vroomy) initRouteGroup(g *RouteGroup) (err error) {
	if g == nil {
		return fmt.Errorf("route group must be non-nil")
	}

	for _, handlerKey := range g.Handlers {
		var h httpserve.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("resolve handler for group %q: %w", g.Name, err)
			return err
		}

		g.HTTPHandlers = append(g.HTTPHandlers, h)
	}

	var (
		match *RouteGroup
		grp   httpserve.Group = newHandlerGroup(v.srv, "", nil)
	)

	if match, err = v.cfg.GetRouteGroup(g.Group); err != nil {
		return err
	} else if match != nil {
		if grp = match.G; grp == nil {
			err = fmt.Errorf("parent group \"%s\" has not yet been initialized", match.Name)
			return err
		}
	}

	g.G = grp.Group(g.HTTPPath, g.HTTPHandlers...)
	return err
}

func (v *Vroomy) initRoutes() (err error) {
	// Set panic func
	v.srv.SetPanic(v.handlePanic)

	for _, r := range v.cfg.Routes {
		if err = v.initRoute(r); err != nil {
			return err
		}

		var (
			match *RouteGroup
			grp   httpserve.Group = newHandlerGroup(v.srv, "", nil)
		)

		if match, err = v.cfg.GetRouteGroup(r.Group); err != nil {
			return err
		} else if match != nil {
			if match.G == nil {
				if err = v.initRouteGroup(match); err != nil {
					return err
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
			return err
		}
	}

	return err
}

func (v *Vroomy) initRoute(r *Route) (err error) {
	if r == nil {
		return fmt.Errorf("route must be non-nil")
	}

	for _, handlerKey := range r.Handlers {
		var h httpserve.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("resolve handler for route %q: %w", r.HTTPPath, err)
			return err
		}

		r.HTTPHandlers = append(r.HTTPHandlers, h)
	}

	return err
}

func (v *Vroomy) setDependencies(pluginKey string, dm dependencyMap) (err error) {
	var pi Plugin
	if pi, err = v.getPlugin(pluginKey); err != nil {
		return err
	}

	rval := reflect.ValueOf(pi)
	if rval.Kind() == reflect.Ptr {
		rval = rval.Elem()
	}

	for depKey, indices := range dm {
		var field reflect.Value
		if field, err = getField(rval, indices); err != nil {
			return fmt.Errorf("dependency %q: %w", depKey, err)
		}

		if err = v.setBackend(field, depKey); err != nil {
			return err
		}
	}

	return pi.Load(v.cfg.Environment)
}

func (v *Vroomy) getPlugin(key string) (pi Plugin, err error) {
	var ok bool
	if pi, ok = v.pm[key]; !ok {
		err = fmt.Errorf("plugin with key of <%s> has not been registered: %w", key, ErrPluginNotLoaded)
		return pi, err
	}

	return pi, err
}

func (v *Vroomy) getReference(key string) (reference interface{}, err error) {
	var pi Plugin
	if pi, err = v.getPlugin(key); err != nil {
		return reference, err
	}

	if reference = pi.Backend(); isNilValue(reflect.ValueOf(reference)) {
		// The provided value isn't an exact match, nor does it match the provided interface
		err = fmt.Errorf("cannot call backend for plugin <%s>, provided value is nil", key)
		return reference, err
	}

	return reference, err
}

func (v *Vroomy) setBackend(backend reflect.Value, key string) (err error) {
	if !backend.IsValid() || !backend.CanSet() {
		return ErrNotAddressable
	}

	var reference interface{}
	if reference, err = v.getReference(key); err != nil {
		return err
	}

	beVal := reflect.ValueOf(reference)
	if err = canSet(backend, beVal); err != nil {
		return err
	}

	backend.Set(beVal)
	return err
}

func (v *Vroomy) loadPlugins() error {
	for index, pluginKey := range v.pluginOrder {
		if err := v.setDependencies(pluginKey, v.dependencies[pluginKey]); err != nil {
			return fmt.Errorf("load plugin %q: %w", pluginKey, err)
		}

		log.Printf("Vroomy: Loaded %s (%d/%d)", pluginKey, index+1, len(v.pluginOrder))
	}

	return nil
}

func (v *Vroomy) closePlugins() error {
	var errs []error
	for i := len(v.initialized) - 1; i >= 0; i-- {
		key := v.initialized[i]
		if closeErr := v.pm[key].Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("close plugin %q: %w", key, closeErr))
		}
	}

	return stderrors.Join(errs...)
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
			return
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
	log.Printf("Vroomy: Panic caught:\n%v\n%s\n\n", in, string(debug.Stack()))
}

// Listen starts the configured HTTP/HTTPS listeners and blocks until a listener
// returns or ctx ends. Cancellation returns ctx.Err() without closing the service.
func (v *Vroomy) Listen(ctx context.Context) (err error) {
	// Initialize error channel
	errC := make(chan error, 2)
	// Listen to HTTP (if needed)
	go v.listenHTTP(errC)
	// Listen to HTTPS (if needed)
	go v.listenHTTPS(errC)

	timer := time.NewTimer(time.Millisecond * 100)

	// Wait for one of the following:
	// - Timer to end, which emits a startup message without proving readiness
	// - Error to come down error channel, which means an error occurred during listening
	// - Context is finished, which means the caller no longer needing this action to continue
	select {
	case <-timer.C:
		v.listenNotification()
	case err = <-errC:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for one of the following:
	// - Error to come down error channel, which means an error occurred during listening
	// - Context is finished, which means the caller no longer needing this action to continue
	select {
	case err = <-errC:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ListenUntilSignal cancels listening on SIGINT, SIGTERM, or SIGABRT, then closes
// the service. It also closes when ctx ends or a listener returns, and suppresses
// context.Canceled while preserving other listen and close errors.
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

// Close closes the HTTP server and plugins in reverse dependency order, then releases
// the registry for reuse. Repeated calls return errors.ErrIsClosed. Plugins must
// tolerate Close after a partially failed Init and support Init again for reuse.
func (v *Vroomy) Close() (err error) {
	if !v.closed.Set(true) {
		return errors.ErrIsClosed
	}

	err = stderrors.Join(v.srv.Close(), v.closePlugins())
	if v.registry != nil {
		v.registry.release()
	}

	return err
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

	log.Printf("Vroomy: %v", msg)
}

// onClose waits for SIGINT, SIGTERM, or SIGABRT and invokes fn.
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

// Register adds a plugin to the process-wide registry under a unique key.
// Register non-nil pointers to structs before constructing a service.
func Register(key string, pi Plugin) error {
	return p.Register(key, pi)
}
