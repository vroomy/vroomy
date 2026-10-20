package vroomy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/vroomy/apex"
)

type handlerResolutionPlugin struct {
	BasePlugin
	arguments []string
	failure   error
}

func (*handlerResolutionPlugin) Direct(ctx *apex.Context) {
	ctx.WriteString(202, "text/plain", "direct")
}

func (p *handlerResolutionPlugin) Factory(arguments ...string) (apex.Handler, error) {
	p.arguments = arguments
	return p.Direct, p.failure
}

func (*handlerResolutionPlugin) Invalid() string { return "invalid signature" }

func (*handlerResolutionPlugin) Middleware(ctx *apex.Context) { ctx.Put("middleware", "resolved") }

func (*handlerResolutionPlugin) ReadMiddleware(ctx *apex.Context) {
	ctx.WriteString(200, "text/plain", ctx.Get("middleware"))
}

func (*handlerResolutionPlugin) Panic(*apex.Context) { panic("test handler panic") }

func TestConfiguredHandlerResolutionAndPanic(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	if err := Register("handlers", &handlerResolutionPlugin{}); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	var previous = log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	var cfg = &Config{Dir: ".", Port: availablePorts(t, 1)[0], IncludeConfig: IncludeConfig{
		Groups: []*RouteGroup{{Name: "group", HTTPPath: "/group", Handlers: []string{"handlers.Middleware"}}},
		Routes: []*Route{
			{Group: "group", HTTPPath: "/ping", Handlers: []string{"handlers.ReadMiddleware"}},
			{HTTPPath: "/panic", Handlers: []string{"handlers.Panic"}},
		},
	}}
	var running = startService(t, cfg)
	var client = &http.Client{Timeout: time.Second}
	for _, route := range []string{"/group/ping", "/panic"} {
		var request, err = http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, route), nil)
		if err != nil {
			t.Fatal(err)
		}

		var response = requestEventually(t, client, request)
		var body = responseBody(t, response)
		if route == "/group/ping" && (response.StatusCode != 200 || body != "resolved") {
			t.Fatalf("configured middleware response = %d %q", response.StatusCode, body)
		}

		if route == "/panic" && response.StatusCode != 500 {
			t.Fatalf("panic status = %d, want 500", response.StatusCode)
		}
	}

	running.cancel()
	select {
	case <-running.done:
	case <-time.After(5 * time.Second):
		t.Fatal("panic test service did not stop")
	}

	if !bytes.Contains(output.Bytes(), []byte("test handler panic")) || !bytes.Contains(output.Bytes(), []byte("handlePanic")) {
		t.Fatalf("panic log lacks cause or stack: %s", output.String())
	}
}

func TestHandlerResolution(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	var cause = errors.New("factory failure")
	var plugin = &handlerResolutionPlugin{}
	if err := Register("handlers", plugin); err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		name      string
		reference string
		failure   error
		wantErr   bool
		cause     error
		arguments []string
	}{
		{name: "direct", reference: "handlers.Direct"},
		{name: "factory", reference: "handlers.Factory(one, two,a=b)", arguments: []string{"one", " two", "a=b"}},
		{name: "factory without parentheses", reference: "handlers.Factory"},
		{name: "factory error", reference: "handlers.Factory", failure: cause, wantErr: true, cause: cause},
		{name: "invalid signature", reference: "handlers.Invalid", wantErr: true, cause: ErrInvalidPluginHandler},
		{name: "missing method", reference: "handlers.Missing", wantErr: true},
		{name: "missing plugin", reference: "missing.Direct", wantErr: true, cause: ErrPluginNotLoaded},
		{name: "malformed reference", reference: "handlers", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin.failure = tt.failure
			plugin.arguments = nil
			var handler, err = getHandler(tt.reference)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getHandler() = %v, want error %t", err, tt.wantErr)
			}

			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Fatalf("error = %v, want cause %v", err, tt.cause)
			}

			if tt.wantErr {
				return
			}

			if !slices.Equal(plugin.arguments, tt.arguments) {
				t.Fatalf("factory arguments = %q, want %q", plugin.arguments, tt.arguments)
			}

			var cfg = &Config{Dir: ".", Port: availablePorts(t, 1)[0], IncludeConfig: IncludeConfig{
				Routes: []*Route{{HTTPPath: "/", HTTPHandlers: []apex.Handler{handler}}},
			}}
			startService(t, cfg)
			var request, requestErr = http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/", cfg.Port), nil)
			if requestErr != nil {
				t.Fatal(requestErr)
			}

			var response = requestEventually(t, &http.Client{Timeout: time.Second}, request)
			if body := responseBody(t, response); response.StatusCode != 202 || body != "direct" {
				t.Fatalf("resolved handler response = %d %q", response.StatusCode, body)
			}
		})
	}
}

func TestHostPolicyResolutionErrors(t *testing.T) {
	var tests = []struct {
		name   string
		plugin Plugin
		closed bool
		cause  error
	}{
		{name: "missing method", plugin: &BasePlugin{}},
		{name: "wrong signature", plugin: &invalidHostPolicyPlugin{}, cause: ErrInvalidHostPolicy},
		{name: "closed registry", closed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			if tt.plugin != nil {
				if err := Register("autocert", tt.plugin); err != nil {
					t.Fatal(err)
				}
			}

			if tt.closed {
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}

			var cfg Config
			var policy, err = cfg.getHostPolicy()
			if policy != nil || err == nil {
				t.Fatalf("getHostPolicy() = %v, %v, want failure", policy == nil, err)
			}

			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Fatalf("error = %v, want cause %v", err, tt.cause)
			}
		})
	}
}

type invalidHostPolicyPlugin struct{ BasePlugin }

func (*invalidHostPolicyPlugin) HostPolicy(context.Context, string) bool { return true }
