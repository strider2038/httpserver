package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const DefaultShutdownTimeout = time.Second

// Server is tiny wrapper for http.Server with graceful shutdown and functional options.
type Server struct {
	*http.Server

	shutdownTimeout time.Duration
}

type OptionFunc func(server *Server)

func TLSConfig(tlsConfig *tls.Config) OptionFunc {
	return func(server *Server) {
		server.TLSConfig = tlsConfig
	}
}

func ReadTimeout(timeout time.Duration) OptionFunc {
	return func(server *Server) {
		server.ReadTimeout = timeout
	}
}

func ReadHeaderTimeout(timeout time.Duration) OptionFunc {
	return func(server *Server) {
		server.ReadHeaderTimeout = timeout
	}
}

func WriteTimeout(timeout time.Duration) OptionFunc {
	return func(server *Server) {
		server.WriteTimeout = timeout
	}
}

func IdleTimeout(timeout time.Duration) OptionFunc {
	return func(server *Server) {
		server.IdleTimeout = timeout
	}
}

func MaxHeaderBytes(maxHeaderBytes int) OptionFunc {
	return func(server *Server) {
		server.MaxHeaderBytes = maxHeaderBytes
	}
}

func TLSNextProto(tlsNextProto map[string]func(*http.Server, *tls.Conn, http.Handler)) OptionFunc {
	return func(server *Server) {
		server.TLSNextProto = tlsNextProto
	}
}

func ConnState(connState func(net.Conn, http.ConnState)) OptionFunc {
	return func(server *Server) {
		server.ConnState = connState
	}
}

func ErrorLog(logger *log.Logger) OptionFunc {
	return func(server *Server) {
		server.ErrorLog = logger
	}
}

func BaseContext(baseContext func(net.Listener) context.Context) OptionFunc {
	return func(server *Server) {
		server.BaseContext = baseContext
	}
}

func ConnContext(connContext func(ctx context.Context, c net.Conn) context.Context) OptionFunc {
	return func(server *Server) {
		server.ConnContext = connContext
	}
}

func ShutdownTimeout(timeout time.Duration) OptionFunc {
	return func(server *Server) {
		server.shutdownTimeout = timeout
	}
}

func New(addr string, handler http.Handler, options ...OptionFunc) *Server {
	server := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}

	for _, setOption := range options {
		setOption(server)
	}
	if server.shutdownTimeout <= 0 {
		server.shutdownTimeout = DefaultShutdownTimeout
	}

	return server
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	done := make(chan struct{}, 1)

	go func() {
		<-ctx.Done()

		shutdownContext, cancel := context.WithTimeout(context.Background(), server.shutdownTimeout)
		defer cancel()

		server.SetKeepAlivesEnabled(false)

		if err := server.Shutdown(shutdownContext); err != nil {
			server.logf("failed to gracefully shutdown the server: %v", err)
		}

		close(done)
	}()

	err := server.Server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("could not listen on %s: %w", server.Addr, err)
	}

	<-done

	return nil
}

func (server *Server) logf(format string, v ...interface{}) {
	if server.ErrorLog != nil {
		server.ErrorLog.Printf(format, v...)
	}
}
