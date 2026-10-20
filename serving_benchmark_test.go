package vroomy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vroomy/apex"
)

type servingPlugin struct {
	BasePlugin
	calls uint64
	body  []byte
}

func (p *servingPlugin) NoContent(ctx *apex.Context) {
	p.calls++
	ctx.WriteNoContent()
}

func (p *servingPlugin) Bytes(ctx *apex.Context) {
	ctx.WriteBytes(200, "text/plain", p.body)
}

func (*servingPlugin) String(ctx *apex.Context) {
	ctx.WriteString(200, "text/plain", "PONG\n")
}

func (*servingPlugin) Middleware(ctx *apex.Context) {
	ctx.Put("middleware", "resolved")
}

func (*servingPlugin) Param(ctx *apex.Context) {
	if ctx.Param("id") != "42" || ctx.Get("middleware") != "resolved" {
		panic("missing route parameter or middleware")
	}

	ctx.WriteNoContent()
}

func (p *servingPlugin) Factory(...string) (apex.Handler, error) {
	return p.NoContent, nil
}

// servingWriter retains header capacity and discards bodies. Requests and writers
// are created before timing so benchmarks measure serving rather than recording.
type servingWriter struct {
	header http.Header
	status int
}

func (w *servingWriter) Header() http.Header { return w.header }

func (w *servingWriter) WriteHeader(status int) { w.status = status }

func (*servingWriter) Write(body []byte) (int, error) { return len(body), nil }

func benchmarkServingRouter(b *testing.B, route *Route) *apex.Router {
	b.Helper()
	var previous = p
	p = newPlugins()
	b.Cleanup(func() { p = previous })
	if err := Register("serving", &servingPlugin{body: []byte("PONG\n")}); err != nil {
		b.Fatal(err)
	}

	var service Vroomy
	if err := service.initRoute(route); err != nil {
		b.Fatal(err)
	}

	// Engine does not expose ServeHTTP. Obtain apex's normal middleware wrapper
	// through its public fallback API, then register it on the measured router.
	var chain apex.Router
	chain.SetNotFound(route.HTTPHandlers...)
	var handler, _, _ = chain.Match(http.MethodGet, "/")
	var router apex.Router
	router.SetNotFound(func(ctx *apex.Context) { ctx.WriteNoContent() })
	if err := router.Handle(http.MethodGet, route.HTTPPath, handler); err != nil {
		b.Fatal(err)
	}

	return &router
}

func BenchmarkPluginServing(b *testing.B) {
	var cases = []struct {
		name     string
		path     string
		request  string
		handlers []string
	}{
		{name: "Direct", path: "/ping", request: "/ping", handlers: []string{"serving.NoContent"}},
		{name: "Factory", path: "/ping", request: "/ping", handlers: []string{"serving.Factory"}},
		{name: "Bytes", path: "/ping", request: "/ping", handlers: []string{"serving.Bytes"}},
		{name: "String", path: "/ping", request: "/ping", handlers: []string{"serving.String"}},
		{name: "MiddlewareParam", path: "/users/:id", request: "/users/42", handlers: []string{"serving.Middleware", "serving.Param"}},
	}
	for _, benchmark := range cases {
		b.Run(benchmark.name, func(b *testing.B) {
			var router = benchmarkServingRouter(b, &Route{HTTPPath: benchmark.path, Handlers: benchmark.handlers})
			var request = httptest.NewRequest(http.MethodGet, benchmark.request, nil)
			var writer = &servingWriter{header: make(http.Header)}
			router.ServeHTTP(writer, request)
			b.ReportAllocs()
			for b.Loop() {
				router.ServeHTTP(writer, request)
			}
		})
	}
}

func BenchmarkRedirectServing(b *testing.B) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		b.Run(method, func(b *testing.B) {
			var listener = newRedirectListener(8443, nil)
			var request = httptest.NewRequest(method, "http://example.com/ping?value=a%3Db", nil)
			var writer = &servingWriter{header: make(http.Header)}
			listener.server.Handler.ServeHTTP(writer, request)
			b.ReportAllocs()
			for b.Loop() {
				listener.server.Handler.ServeHTTP(writer, request)
			}
		})
	}
}
