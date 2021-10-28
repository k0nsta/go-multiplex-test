package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	BindAddr string
	Handler  http.Handler
}

type APIServer struct {
	srv    *http.Server
	config *Config
}

// New construct APIServer
func New(cfg *Config) *APIServer {
	return &APIServer{
		config: cfg,
		srv: &http.Server{
			ReadHeaderTimeout: time.Second * 5,
			Handler:           cfg.Handler,
			Addr:              cfg.BindAddr,
		},
	}
}

func (s *APIServer) Serve() <-chan error {
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", s.config.BindAddr)
	if err != nil {
		errChan <- err

		return errChan
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT,
	)

	go func() {
		<-ctx.Done()
		log.Printf("[WARN][%s]: shutdown signal has recieved\n", time.Now().Format(time.RFC3339))
		ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)

		defer func() {
			stop()
			cancel()
			close(errChan)
		}()

		if err := s.shutdown(ctxTimeout); err != nil {
			errChan <- err
		}
		log.Printf("[WARN][%s]: shutdown has completed\n", time.Now().Format(time.RFC3339))
	}()

	go func() {
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	log.Printf("[INFO][%s]: server running: %s\n", time.Now().Format(time.RFC3339), s.srv.Addr)

	return errChan
}

func (s *APIServer) shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
