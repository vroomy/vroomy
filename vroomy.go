package vroomy

import (
	"context"
	"fmt"
	"os"
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
func New(configLocation string) (sp *Vroomy, err error) {
	var cfg *config.Config
	if cfg, err = config.NewConfig(configLocation); err != nil {
		return
	}

	if _, ok := cfg.Environment["dataDir"]; !ok {
		// Default if not set elsewhere
		cfg.Environment["dataDir"] = "data"
	}

	return NewWithConfig(cfg)
}

// NewWithConfig will return a new instance of service with a provided config
func NewWithConfig(cfg *config.Config) (vp *Vroomy, err error) {
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
	pluginList := plugins.Loaded()

	if err = v.initPlugins(pluginList); err != nil {
		err = fmt.Errorf("error loading plugins: %v", err)
		return
	}

	if err = v.loadPlugins(pluginList); err != nil {
		err = fmt.Errorf("error initializing plugins: %v", err)
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

	// TODO: Move this to docs/testing only?
	if err = v.initRouteExamples(); err != nil {
		err = fmt.Errorf("error initializing routes: %v", err)
		return
	}

	vp = &v
	return
}

// Vroomy manages the web service
type Vroomy struct {
	cfg *config.Config
	srv *httpserve.Serve

	out *scribe.Scribe

	// Closed state
	closed atoms.Bool
}

func (v *Vroomy) initPlugins(pluginList map[string]plugins.Plugin) (err error) {
	// Call Init(flags, env) for each initialized plugin
	for pluginKey, plugin := range pluginList {
		if err = plugin.Init(v.cfg.Environment); err != nil {
			return
		}

		v.out.Notificationf("Initialized %s", pluginKey)
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

		if err = v.initGroup(group); err != nil {
			return
		}
	}

	return
}

func (v *Vroomy) initGroup(g *config.Group) (err error) {
	for _, handlerKey := range g.Handlers {
		var h common.Handler
		if h, err = getHandler(handlerKey); err != nil {
			return
		}

		g.HTTPHandlers = append(g.HTTPHandlers, h)
	}

	var (
		match *config.Group
		grp   common.Group = v.srv
	)

	if match, err = v.cfg.GetGroup(g.Group); err != nil {
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
			match *config.Group
			grp   common.Group = v.srv
		)

		if match, err = v.cfg.GetGroup(r.Group); err != nil {
			return
		} else if match != nil {
			if match.G == nil {
				if err = v.initGroup(match); err != nil {
					return
				}
			}

			grp = match.G
		}

		var fn func(string, ...common.Handler)
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

func (v *Vroomy) initRoute(r *config.Route) (err error) {
	fmt.Println("initing route", r)
	for _, handlerKey := range r.Handlers {
		var h common.Handler
		if h, err = getHandler(handlerKey); err != nil {
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

func (v *Vroomy) initRouteExamples() (err error) {
	v.cfg.ExampleResponses = make(map[string]*config.Response)
	var needsParentRes = []*config.Response{}
	for _, res := range v.cfg.Responses {
		v.cfg.ExampleResponses[res.Name] = res
		if len(strings.TrimSpace(res.Parent)) > 0 {
			needsParentRes = append(needsParentRes, res)
		}
	}

	for _, res := range needsParentRes {
		if _, ok := v.cfg.ExampleResponses[res.Parent]; ok {
			res.InheritFrom(v.cfg.ExampleResponses)
		} else {
			v.out.Warningf("Unable to find parent (%s) for response: %s", res.Parent, res.Name)
		}
	}

	v.cfg.ExampleRequests = make(map[string]*config.Request)
	var needsParentReq = []*config.Request{}
	for _, req := range v.cfg.Requests {
		v.cfg.ExampleRequests[req.Name] = req

		if len(req.Group) > 0 {
			var g *config.Group
			if g, err = v.cfg.GetGroup(req.Group); err != nil {
				v.out.Warningf("Unable to find group (%s) for request: ", req.Name)
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
			if res, ok := v.cfg.ExampleResponses[resName]; ok {
				req.ResponseExamples = append(req.ResponseExamples, res)
			} else {
				v.out.Warningf("Unable to find response (%s) for request: %s", resName, req.Name)
			}
		}
	}

	for _, req := range needsParentReq {
		if _, ok := v.cfg.ExampleRequests[req.Parent]; ok {
			req.InheritFrom(v.cfg.ExampleRequests)
		} else {
			v.out.Warningf("Unable to find parent (%s) for request: %s", req.Parent, req.Name)
		}
	}

	return
}

func (v *Vroomy) loadPlugins(pluginList map[string]plugins.Plugin) (err error) {
	// Call Init(flags, env) for each initialized plugin
	for pluginKey, plugin := range pluginList {
		if err = plugin.Load(); err != nil {
			return
		}

		v.out.Notificationf("Loaded %s", pluginKey)
	}

	return
}

func (v *Vroomy) getHTTPListener() (l listener) {
	if v.cfg.TLSPort > 0 {
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

	if len(v.cfg.TLSDir) == 0 {
		// Cannot serve TLS without a tls directory, send error down channel and return
		errC <- ErrInvalidTLSDirectory
		return
	}

	// Attempt to listen to HTTPS with the configured tls port and directory
	errC <- v.srv.ListenTLS(v.cfg.TLSPort, v.cfg.TLSDir)
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

	select {
	case err = <-errC:
		return
	case <-ctx.Done():
		return ctx.Err()
	}
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
	return errs.Err()
}
