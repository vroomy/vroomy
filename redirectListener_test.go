package vroomy

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type failingResponseWriter struct {
	header http.Header
	status int
	cause  error
}

func (w *failingResponseWriter) Header() http.Header { return w.header }

func (w *failingResponseWriter) WriteHeader(status int) { w.status = status }

func (w *failingResponseWriter) Write([]byte) (int, error) { return 0, w.cause }

func TestRedirectResponseWriteErrors(t *testing.T) {
	var cause = errors.New("client disconnected")
	for _, callback := range []bool{true, false} {
		t.Run(fmt.Sprint(callback), func(t *testing.T) {
			var output bytes.Buffer
			var previous = log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			var reported error
			var onError func(error)
			if callback {
				onError = func(err error) { reported = err }
			}

			var listener = newRedirectListener(443, onError)
			var writer = &failingResponseWriter{header: make(http.Header), cause: cause}
			listener.server.Handler.ServeHTTP(writer, httptest.NewRequest("POST", "http://example.com/", nil))
			if writer.status != 404 || writer.header.Get("Content-Type") != "text/plain" {
				t.Fatalf("response = %d %v", writer.status, writer.header)
			}

			if callback && !errors.Is(reported, cause) {
				t.Fatalf("callback error = %v, want %v", reported, cause)
			}

			if !callback && !strings.Contains(output.String(), cause.Error()) {
				t.Fatalf("write failure not logged: %q", output.String())
			}
		})
	}
}

func TestRedirectListenerStartupStates(t *testing.T) {
	t.Run("closed before start", func(t *testing.T) {
		var listener = newRedirectListener(443, nil)
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}

		if err := listener.Listen(0); !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Listen() = %v, want server closed", err)
		}
	})
	t.Run("port already occupied", func(t *testing.T) {
		var occupied, err = net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			if closeErr := occupied.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		})
		var listener = newRedirectListener(443, nil)
		if err = listener.Listen(uint16(occupied.Addr().(*net.TCPAddr).Port)); err == nil {
			t.Fatal("Listen accepted occupied port")
		}

		if err = listener.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("second start and h2c", func(t *testing.T) {
		t.Setenv("USE_HTTP2_RAW", "false")
		var listener = newRedirectListener(443, nil)
		var done = make(chan error, 1)
		go func() { done <- listener.Listen(0) }()
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Error(err)
			}

			select {
			case err := <-done:
				if !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("Listen() = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Error("redirect listener did not stop")
			}
		})
		var address string
		var deadline = time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			listener.mu.Lock()
			if listener.listener != nil {
				address = listener.listener.Addr().String()
			}

			listener.mu.Unlock()
			if address != "" {
				break
			}

			time.Sleep(time.Millisecond)
		}

		if address == "" {
			t.Fatal("redirect listener did not start")
		}

		if err := listener.Listen(0); err == nil || !strings.Contains(err.Error(), "already started") {
			t.Fatalf("second Listen() = %v", err)
		}

		var transport = &http.Transport{Protocols: new(http.Protocols)}
		transport.Protocols.SetUnencryptedHTTP2(true)
		t.Cleanup(transport.CloseIdleConnections)
		var client = &http.Client{Transport: transport, Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		var response, err = client.Get("http://" + address + "/ping?x=a%3Db")
		if err != nil {
			t.Fatal(err)
		}

		responseBody(t, response)
		if response.StatusCode != 301 || response.ProtoMajor != 2 {
			t.Fatalf("redirect = %d %s, want 301 over h2c", response.StatusCode, response.Proto)
		}
	})
}
