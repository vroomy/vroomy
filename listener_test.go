package vroomy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vroomy/apex"
	"golang.org/x/net/http2"
)

func writeTestCertificate(t *testing.T, dir, host string) ([]byte, []byte, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: host},
		DNSNames: []string{host}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(90 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	for name, contents := range map[string][]byte{"server.crt": certificate, "server.key": privateKey} {
		if err = os.WriteFile(filepath.Join(dir, name), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificate) {
		t.Fatal("could not trust test certificate")
	}

	return certificate, privateKey, roots
}

func TestTLSRedirectAndShutdown(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	ports := availablePorts(t, 2)
	certDir := t.TempDir()
	_, _, roots := writeTestCertificate(t, certDir, "localhost")
	cfg := &Config{Dir: ".", Port: ports[0], TLSPort: ports[1], TLSDir: certDir}
	cfg.Routes = []*Route{{HTTPPath: "/ping", HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.WriteString(200, "text/plain", "PONG") }}}}
	running := startService(t, cfg)
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "OPTIONS", "PATCH"} {
		request, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d/ping?value=a%%3Db", cfg.Port), nil)
		if err != nil {
			t.Fatal(err)
		}

		response := requestEventually(t, client, request)
		responseBody(t, response)
		if method == "GET" {
			want := fmt.Sprintf("https://127.0.0.1:%d/ping?value=a%%3Db", cfg.TLSPort)
			if response.StatusCode != 301 || response.Header.Get("Location") != want {
				t.Errorf("redirect = %d %q, want 301 %q", response.StatusCode, response.Header.Get("Location"), want)
			}
		} else if response.StatusCode != 404 {
			t.Errorf("%s = %d, want 404", method, response.StatusCode)
		}
	}

	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: "localhost"}, ForceAttemptHTTP2: true}
	t.Cleanup(transport.CloseIdleConnections)
	tlsClient := &http.Client{Transport: transport, Timeout: time.Second}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/ping", cfg.TLSPort), nil)
	if err != nil {
		t.Fatal(err)
	}

	response := requestEventually(t, tlsClient, request)
	if body := responseBody(t, response); response.StatusCode != 200 || body != "PONG" || response.ProtoMajor != 2 {
		t.Fatalf("TLS response = %d %q %s, want 200 PONG over HTTP/2", response.StatusCode, body, response.Proto)
	}

	if err = running.service.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-running.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not stop Listen")
	}

	assertPortsReleased(t, ports...)
}

func TestH2CProcessEnvironment(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	t.Setenv("USE_HTTP2_RAW", "false")
	cfg := &Config{Dir: ".", Port: availablePorts(t, 1)[0]}
	cfg.Routes = []*Route{{HTTPPath: "/ping", HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.WriteString(200, "text/plain", "PONG") }}}}
	startService(t, cfg)
	transport := &http2.Transport{AllowHTTP: true, DialTLSContext: func(ctx context.Context, network, address string, _ *tls.Config) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: time.Second}
	request, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/ping", cfg.Port), nil)
	if err != nil {
		t.Fatal(err)
	}

	response := requestEventually(t, client, request)
	if body := responseBody(t, response); body != "PONG" || response.ProtoMajor != 2 {
		t.Fatalf("h2c response = %q %s", body, response.Proto)
	}
}

func TestListenImmediateCancellation(t *testing.T) {
	for i := 0; i < 10; i++ {
		isolateRegistry(t)
		preserveWorkingDirectory(t)
		cfg := &Config{Dir: ".", Port: availablePorts(t, 1)[0]}
		service, err := NewWithConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err = service.Listen(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Listen() = %v, want cancellation", err)
		}

		assertPortsReleased(t, cfg.Port)
	}
}

func TestAutoCertCachedCertificate(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	cacheDir := t.TempDir()
	certificate, key, roots := writeTestCertificate(t, t.TempDir(), "cached.example")
	if err := os.WriteFile(filepath.Join(cacheDir, "cached.example"), append(key, certificate...), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{Dir: ".", TLSPort: availablePorts(t, 1)[0], IncludeConfig: IncludeConfig{
		AutoCertDir: cacheDir, AutoCertHosts: []string{"cached.example"},
		Routes: []*Route{{HTTPPath: "/ping", HTTPHandlers: []apex.Handler{func(ctx *apex.Context) { ctx.WriteString(200, "text/plain", "PONG") }}}},
	}}
	running := startService(t, cfg)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: "cached.example"}, ForceAttemptHTTP2: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: time.Second}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/ping", cfg.TLSPort), nil)
	if err != nil {
		t.Fatal(err)
	}

	response := requestEventually(t, client, request)
	if body := responseBody(t, response); body != "PONG" || response.ProtoMajor != 2 {
		t.Fatalf("autocert response = %q %s", body, response.Proto)
	}

	// A rejected hostname must fail before any certificate issuance attempt.
	connection, handshakeErr := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp", fmt.Sprintf("127.0.0.1:%d", cfg.TLSPort), &tls.Config{RootCAs: roots, ServerName: "rejected.example"})
	if handshakeErr == nil {
		if closeErr := connection.Close(); closeErr != nil {
			t.Error(closeErr)
		}

		t.Fatal("unlisted autocert hostname was accepted")
	}

	running.cancel()
	select {
	case <-running.done:
	case <-time.After(5 * time.Second):
		t.Fatal("autocert shutdown timed out")
	}

	assertPortsReleased(t, cfg.TLSPort)
}

func TestListenUntilSignalParentCancellation(t *testing.T) {
	isolateRegistry(t)
	service, err := NewWithConfig(&Config{Dir: "."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = service.ListenUntilSignal(ctx); err != nil {
		t.Fatalf("normal shutdown = %v", err)
	}

	if !service.closed.Get() {
		t.Fatal("service was not closed")
	}
}

func TestListenUntilSignalPreservesCleanupError(t *testing.T) {
	isolateRegistry(t)
	var events []string
	cause := errors.New("close failure")
	provider := &lifecyclePlugin{events: &events, key: "provider", closeErr: cause}
	if err := Register("provider", &lifecycleProvider{plugin: provider}); err != nil {
		t.Fatal(err)
	}

	service, err := NewWithConfig(&Config{Dir: "."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = service.ListenUntilSignal(ctx)
	if !errors.Is(err, cause) || errors.Is(err, context.Canceled) {
		t.Fatalf("shutdown error = %v, want cleanup cause without cancellation", err)
	}
}

func TestFailedTLSStartupReleasesHTTP(t *testing.T) {
	isolateRegistry(t)
	ports := availablePorts(t, 2)
	service, err := NewWithConfig(&Config{Dir: ".", Port: ports[0], TLSPort: ports[1], TLSDir: filepath.Join(t.TempDir(), "missing")})
	if err != nil {
		t.Fatal(err)
	}

	if err = service.Listen(context.Background()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed TLS startup = %v, want filesystem cause", err)
	}

	assertPortsReleased(t, ports...)
}
