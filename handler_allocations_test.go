package vroomy

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/vroomy/apex"
)

type resolvedReceiver struct {
	calls atomic.Uint64
	name  string
}

func (r *resolvedReceiver) Promoted(ctx *apex.Context) {
	r.calls.Add(1)
	ctx.Put("receiver", r.name)
}

type resolvedReceiverPlugin struct {
	BasePlugin
	*resolvedReceiver
}

func (p *resolvedReceiverPlugin) Pointer(ctx *apex.Context) {
	p.calls.Add(1)
	ctx.Put("receiver", p.name)
}

func (p resolvedReceiverPlugin) Value(ctx *apex.Context) {
	p.calls.Add(1)
	ctx.Put("receiver", p.name)
}

func TestResolvedHandlerAllocationsAndReceivers(t *testing.T) {
	isolateRegistry(t)
	var first = &resolvedReceiverPlugin{resolvedReceiver: &resolvedReceiver{name: "first"}}
	var second = &resolvedReceiverPlugin{resolvedReceiver: &resolvedReceiver{name: "second"}}
	for key, plugin := range map[string]Plugin{"first": first, "second": second} {
		if err := Register(key, plugin); err != nil {
			t.Fatal(err)
		}
	}

	for _, method := range []string{"Pointer", "Value", "Promoted"} {
		t.Run(method, func(t *testing.T) {
			// Retain both handlers before executing either to catch receiver reuse.
			var firstHandler, err = getHandler("first." + method)
			if err != nil {
				t.Fatal(err)
			}

			var secondHandler apex.Handler
			if secondHandler, err = getHandler("second." + method); err != nil {
				t.Fatal(err)
			}

			first.calls.Store(0)
			second.calls.Store(0)
			for key, handler := range map[string]apex.Handler{"first": firstHandler, "second": secondHandler} {
				var ctx apex.Context
				// Measure dispatch without pool misses, including the race detector's
				// intentional random sync.Pool evictions.
				var allocations = testing.AllocsPerRun(1000, func() { handler(&ctx) })
				if allocations != 0 {
					t.Errorf("resolved handler allocates %v times per request, want 0", allocations)
				}

				if ctx.Get("receiver") != key {
					t.Errorf("receiver = %q, want %q", ctx.Get("receiver"), key)
				}
			}

			if first.calls.Load() != 1001 || second.calls.Load() != 1001 {
				t.Fatalf("receiver calls = (%d, %d), want (1001, 1001)", first.calls.Load(), second.calls.Load())
			}
		})
	}
}

func TestResolvedHandlerConcurrentRequests(t *testing.T) {
	isolateRegistry(t)
	var plugin = &resolvedReceiverPlugin{resolvedReceiver: &resolvedReceiver{name: "concurrent"}}
	if err := Register("receiver", plugin); err != nil {
		t.Fatal(err)
	}

	var handler, err = getHandler("receiver.Pointer")
	if err != nil {
		t.Fatal(err)
	}

	var router apex.Router
	router.SetNotFound(handler, func(ctx *apex.Context) {
		if ctx.Get("receiver") != "concurrent" {
			t.Error("handler did not receive the request context")
		}

		ctx.WriteNoContent()
	})
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			var request = httptest.NewRequest(http.MethodGet, "/", nil)
			var writer = &servingWriter{header: make(http.Header)}
			for range 1000 {
				router.ServeHTTP(writer, request)
			}
		}()
	}

	workers.Wait()
	if plugin.calls.Load() != 8000 {
		t.Fatalf("receiver calls = %d, want 8000", plugin.calls.Load())
	}
}
