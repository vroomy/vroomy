package vroomy

import (
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	"github.com/gdbu/atoms"
	"github.com/gdbu/errors"
	"github.com/vroomy/apex"
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
	v.closeDone = make(chan struct{})
	if cfg.TLSPort > 0 && !cfg.AllowNonTLS {
		v.redirect = newRedirectListener(cfg.TLSPort, cfg.ErrorLogger)
	}

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
	if err = v.validatePlugins(); err != nil {
		return nil, err
	}

	if v.dependencies, err = makeDependenciesMap(v.pm); err != nil {
		return nil, err
	}

	if err = v.dependencies.Validate(); err != nil {
		return nil, err
	}

	if err = v.dependencies.Load(v.appendPluginOrder); err != nil {
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

	v.srv = apex.New()
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
	srv *apex.Engine

	pm           map[string]Plugin
	registry     *Plugins
	dependencies dependenciesMap
	pluginOrder  []string
	initialized  []string

	redirect  *redirectListener
	listening atoms.Bool
	closeDone chan struct{}
	closeErr  error

	// Closed state
	closed atoms.Bool
}

func (v *Vroomy) validatePlugins() (err error) {
	for key, plugin := range v.pm {
		if err = validatePlugin(plugin); err != nil {
			return fmt.Errorf("validate plugin %q: %w", key, err)
		}
	}

	return nil
}

func (v *Vroomy) appendPluginOrder(key string, _ dependencyMap) error {
	v.pluginOrder = append(v.pluginOrder, key)
	return nil
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

	return nil
}

func (v *Vroomy) initGroups() (err error) {
	if len(v.cfg.Groups) == 0 {
		return nil
	}

	for _, group := range v.cfg.Groups {
		if err = v.initRouteGroup(group); err != nil {
			return err
		}
	}

	return nil
}

func (v *Vroomy) initRouteGroup(g *RouteGroup) (err error) {
	if g == nil {
		return fmt.Errorf("route group must be non-nil")
	}

	for _, handlerKey := range g.Handlers {
		var h apex.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("resolve handler for group %q: %w", g.Name, err)
			return err
		}

		g.HTTPHandlers = append(g.HTTPHandlers, h)
	}

	var (
		match *RouteGroup
		grp   apex.Group = newHandlerGroup(v.srv, "", nil)
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
	return nil
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
			grp   apex.Group = newHandlerGroup(v.srv, "", nil)
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

		var fn func(string, ...apex.Handler) error
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

	return nil
}

func (v *Vroomy) initRoute(r *Route) (err error) {
	if r == nil {
		return fmt.Errorf("route must be non-nil")
	}

	for _, handlerKey := range r.Handlers {
		var h apex.Handler
		if h, err = getHandler(handlerKey); err != nil {
			err = fmt.Errorf("resolve handler for route %q: %w", r.HTTPPath, err)
			return err
		}

		r.HTTPHandlers = append(r.HTTPHandlers, h)
	}

	return nil
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

	return pi, nil
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

	return reference, nil
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
	return nil
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

func (v *Vroomy) listenHTTP(errC chan error) {
	if v.redirect != nil {
		errC <- v.redirect.Listen(v.cfg.Port)
		return
	}

	errC <- v.srv.Listen(v.cfg.Port)
}

func (v *Vroomy) listenHTTPS(errC chan error) {
	switch {
	case v.cfg.hasTLSDir():
		errC <- v.srv.ListenTLS(v.cfg.TLSPort, v.cfg.TLSDir)
	case v.cfg.hasAutoCert():
		ac, err := v.cfg.autoCertConfig()
		if err != nil {
			errC <- err
			return
		}

		errC <- v.srv.ListenAutoCertTLS(v.cfg.TLSPort, ac)
	default:
		errC <- ErrInvalidTLSDirectory
	}
}

func (v *Vroomy) handlePanic(in interface{}) {
	log.Printf("Vroomy: Panic caught:\n%v\n%s\n\n", in, string(debug.Stack()))
}

// Listen runs configured listeners until ctx ends, a listener fails, or Close
// is called. Every exit closes listeners and plugins and joins listener goroutines.
// Cancellation preserves ctx.Err(). A service can only listen once.
func (v *Vroomy) Listen(ctx context.Context) (err error) {
	if v.closed.Get() {
		return errors.ErrIsClosed
	}

	if !v.listening.Set(true) {
		return fmt.Errorf("service has already started listening")
	}

	var listenerCount int
	errC := make(chan error, 2)
	if v.cfg.Port > 0 {
		listenerCount++
		go v.listenHTTP(errC)
	}

	if v.cfg.TLSPort > 0 {
		listenerCount++
		go v.listenHTTPS(errC)
	}

	select {
	case err = <-errC:
		listenerCount--
	case <-ctx.Done():
		err = ctx.Err()
	case <-v.closeDone:
	}

	if stderrors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	closeErr := v.Close()
	if stderrors.Is(closeErr, errors.ErrIsClosed) {
		closeErr = v.closeErr
	}

	err = stderrors.Join(err, closeErr)

	// Buffered results allow both listeners to return during shutdown.
	for ; listenerCount > 0; listenerCount-- {
		listenerErr := <-errC
		if !stderrors.Is(listenerErr, http.ErrServerClosed) {
			err = stderrors.Join(err, listenerErr)
		}
	}

	return err
}

// ListenUntilSignal stops on SIGINT, SIGTERM, or SIGABRT and releases signal
// notification when listening ends. Context cancellation is normal shutdown;
// other listener and cleanup errors are preserved.
func (v *Vroomy) ListenUntilSignal(ctx context.Context) error {
	signalContext, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	defer stop()
	err := v.Listen(signalContext)
	if signalContext.Err() == context.Canceled {
		// Remove only cancellation; retain any other joined shutdown errors.
		return withoutCancellation(err)
	}

	return err
}

// Port will return the current HTTP port
func (v *Vroomy) Port() uint16 {
	return v.cfg.Port
}

// TLSPort will return the current HTTPS port
func (v *Vroomy) TLSPort() uint16 {
	return v.cfg.TLSPort
}

// Close closes listeners and plugins in reverse dependency order, then releases
// the registry for reuse. Repeated calls return errors.ErrIsClosed. Plugins must
// tolerate Close after a partially failed Init and support Init again for reuse.
func (v *Vroomy) Close() (err error) {
	if !v.closed.Set(true) {
		<-v.closeDone
		return errors.ErrIsClosed
	}

	defer close(v.closeDone)
	var redirectErr error
	if v.redirect != nil {
		redirectErr = v.redirect.Close()
	}

	err = stderrors.Join(v.srv.Close(), redirectErr, v.closePlugins())
	v.closeErr = err
	if v.registry != nil {
		v.registry.release()
	}

	return err
}

// Register adds a plugin to the process-wide registry under a unique key.
// Register non-nil pointers to structs before constructing a service.
func Register(key string, pi Plugin) error {
	return p.Register(key, pi)
}
