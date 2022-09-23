package internalhttp

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/pkg/errors"
)

//go:generate mockgen -destination mocks/server_mocks.go -source server.go -package servermocks
type Serverer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type Application interface {
	CreateEvent(ctx context.Context, id, title string) error
}

type Server struct {
	Logg   app.Logger
	App    Application
	Server Serverer

	ShutdownTimeout time.Duration
}

func NewServer(
	logger app.Logger,
	app Application,
	host, port string,
	readTimeout, writeTimeout, shutDownTimeout time.Duration,
) *Server {
	mux := http.NewServeMux()
	mux.Handle("/hello", loggingMiddleware(logger, HelloHandler{}))

	server := &http.Server{
		Addr:              net.JoinHostPort(host, port),
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		Handler:           mux,
		ReadHeaderTimeout: readTimeout,
	}
	return &Server{
		Logg:   logger,
		App:    app,
		Server: server,

		ShutdownTimeout: shutDownTimeout,
	}
}

func (s *Server) Start(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		errChan <- s.Server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		s.Logg.Info().Msg("Stop server by signal")
		return nil
	case err := <-errChan:
		if errors.Is(err, http.ErrServerClosed) {
			s.Logg.Info().Msg("Server is shutdowning...")
			return nil
		}
		return errors.Wrap(err, "Got unexpected error while starting server")
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.Logg.Info().Msg("Start stopping server...")
	ctx, cancel := context.WithTimeout(ctx, s.ShutdownTimeout)
	defer cancel()
	err := s.Server.Shutdown(ctx)
	s.Logg.Info().Err(err).Msg("Server is shutdown")
	return err
}
