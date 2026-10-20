package vroomy

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

func newRedirectListener(tlsPort uint16, onError func(error)) *redirectListener {
	var listener redirectListener
	listener.server = &http.Server{
		ReadTimeout:                  30 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  90 * time.Second,
		MaxHeaderBytes:               4096,
		DisableGeneralOptionsHandler: true,
	}
	listener.server.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.Header().Set("Content-Type", "text/plain")
			writer.WriteHeader(http.StatusNotFound)
			if _, writeErr := writer.Write([]byte("404, not found")); writeErr != nil {
				if onError != nil {
					onError(writeErr)
				} else {
					log.Printf("Vroomy: redirect response: %v", writeErr)
				}
			}

			return
		}

		destination := *request.URL
		hostURL := url.URL{Host: request.Host}
		destination.Scheme = "https"
		destination.Host = net.JoinHostPort(hostURL.Hostname(), strconv.Itoa(int(tlsPort)))
		writer.Header().Set("Location", destination.String())
		writer.WriteHeader(http.StatusMovedPermanently)
	})
	return &listener
}

// redirectListener owns its server and socket, including Close before Serve starts.
type redirectListener struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	closed   bool
}

func (l *redirectListener) Listen(port uint16) error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return http.ErrServerClosed
	}

	if l.listener != nil {
		l.mu.Unlock()
		return fmt.Errorf("redirect listener already started")
	}

	if _, enabled := os.LookupEnv("USE_HTTP2_RAW"); enabled {
		// Enable unencrypted HTTP/2 support directly
		l.server.Protocols = new(http.Protocols)
		l.server.Protocols.SetHTTP1(true)
		l.server.Protocols.SetUnencryptedHTTP2(true)
	}

	var err error
	l.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	listener := l.listener
	l.mu.Unlock()
	if err != nil {
		return fmt.Errorf("listen for redirects on port %d: %w", port, err)
	}

	return l.server.Serve(listener)
}

func (l *redirectListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	serverErr := l.server.Close()
	var listenerErr error
	if l.listener != nil {
		listenerErr = l.listener.Close()
		if errors.Is(listenerErr, net.ErrClosed) {
			listenerErr = nil
		}
	}

	return errors.Join(serverErr, listenerErr)
}
