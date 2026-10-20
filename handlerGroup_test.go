package vroomy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vroomy/apex"
)

func availablePorts(t *testing.T, count int) []uint16 {
	t.Helper()
	var listeners []net.Listener
	var ports []uint16
	for i := 0; i < count; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}

		listeners = append(listeners, listener)
		ports = append(ports, uint16(listener.Addr().(*net.TCPAddr).Port))
	}

	for _, listener := range listeners {
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
	}

	return ports
}

type runningService struct {
	service *Vroomy
	cancel  context.CancelFunc
	done    chan struct{}
	err     error
}

func startService(t *testing.T, cfg *Config) *runningService {
	t.Helper()
	service, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	running := &runningService{service: service, cancel: cancel, done: make(chan struct{})}
	go func() {
		running.err = service.Listen(ctx)
		if !service.closed.Get() {
			running.err = errors.Join(running.err, service.Close())
		}

		close(running.done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-running.done:
			if running.err != nil && !errors.Is(running.err, context.Canceled) {
				t.Errorf("Listen() = %v", running.err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Listen did not release its goroutines")
		}
	})
	return running
}

func requestEventually(t *testing.T, client *http.Client, request *http.Request) *http.Response {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		response, requestErr := client.Do(request.Clone(request.Context()))
		if requestErr == nil {
			return response
		}

		lastErr = requestErr
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("request %s: %v", request.URL, lastErr)
	return nil
}

func responseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(response.Body)
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Error(closeErr)
	}

	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func assertPortsReleased(t *testing.T, ports ...uint16) {
	t.Helper()
	for _, port := range ports {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Errorf("port %d retained: %v", port, err)
			continue
		}

		if err = listener.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestConfiguredHandlerOwnership(t *testing.T) {
	for _, capacity := range []int{3, 4} {
		t.Run(fmt.Sprintf("capacity_%d", capacity), func(t *testing.T) {
			isolateRegistry(t)
			preserveWorkingDirectory(t)
			middleware := make([]apex.Handler, 3, capacity)
			for i := range middleware {
				middleware[i] = func(ctx *apex.Context) { ctx.Put("order", ctx.Get("order")+"P") }
			}

			cfg := &Config{Dir: ".", Port: availablePorts(t, 1)[0]}
			cfg.Groups = []*RouteGroup{
				{Name: "parent", HTTPPath: "/parent", HTTPHandlers: middleware},
				{Name: "a", Group: "parent", HTTPPath: "/a", HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.Put("order", ctx.Get("order")+"A") }}},
				{Name: "b", Group: "parent", HTTPPath: "/b", HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.Put("order", ctx.Get("order")+"B") }}},
			}
			for _, method := range []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"} {
				for _, group := range []string{"parent", "a", "b"} {
					for _, suffix := range []string{"first", "second"} {
						label := suffix
						cfg.Routes = append(cfg.Routes, &Route{Group: group, Method: method, HTTPPath: "/" + suffix, HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.WriteString(200, "text/plain", ctx.Get("order")+label) }}})
					}
				}
			}

			running := startService(t, cfg)
			// Borrowed config slices must not remain mutable views of registered chains.
			middleware[0] = func(ctx *apex.Context) { ctx.Put("order", "corrupted") }
			client := &http.Client{Timeout: time.Second}
			for _, route := range cfg.Routes {
				prefix := "/parent"
				want := "PPP"
				if route.Group != "parent" {
					prefix += "/" + route.Group
					want += strings.ToUpper(route.Group)
				}

				want += strings.TrimPrefix(route.HTTPPath, "/")
				request, err := http.NewRequest(route.Method, fmt.Sprintf("http://127.0.0.1:%d%s%s", cfg.Port, prefix, route.HTTPPath), nil)
				if err != nil {
					t.Fatal(err)
				}

				response := requestEventually(t, client, request)
				if body := responseBody(t, response); response.StatusCode != 200 || body != want {
					t.Errorf("%s %s = %d %q, want 200 %q", route.Method, request.URL.Path, response.StatusCode, body, want)
				}
			}

			running.cancel()
			select {
			case <-running.done:
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown timed out")
			}

			assertPortsReleased(t, cfg.Port)
		})
	}
}
