package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -destination server_mocks.go -source server.go -package server
type Application interface {
	desc.EventServiceServer
}

type HTTPServerer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type GRPCServerer interface {
	grpc.ServiceRegistrar
	Serve(listener net.Listener) error
	GracefulStop()
}

type Server struct {
	Logg     app.Logger
	App      Application
	HTTPHost string
	GRPCHost string

	ShutdownTimeout time.Duration

	httpServer  HTTPServerer
	grpcServer  GRPCServerer
	tcpListener net.Listener
}

func NewServer(
	ctx context.Context,
	logger app.Logger,
	app desc.EventServiceServer,
	mux http.Handler,
	httpHost, grpcHost string,
	readTimeout, writeTimeout, shutDownTimeout time.Duration,
) *Server {
	httpServer := &http.Server{
		Addr:              httpHost,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		Handler:           mux,
		ReadHeaderTimeout: readTimeout,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout:          time.Second * 10,
			MaxConnectionAge: time.Second * 10,
		}),
		grpc.ChainUnaryInterceptor(
			func(
				ctx context.Context,
				req interface{},
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (resp interface{}, err error) {
				startTime := time.Now()
				remoteAddr := "unknown"
				if p, ok := peer.FromContext(ctx); ok {
					remoteAddr = p.Addr.String()
				}
				userAgent := "unknown"
				if md, ok := metadata.FromIncomingContext(ctx); ok {
					userAgent = strings.Join(md.Get("user-agent"), ",")
				}

				resp, err = handler(ctx, req)

				statusCode, _ := status.FromError(err)
				logger.Debug().Msgf("%s [%s] %s %s %s %d %s",
					remoteAddr,
					time.Now().UTC().Format("02/Jan/2006:15:04:05 -0700"),
					info.FullMethod,
					"gRPC",
					statusCode.Code().String(),
					time.Since(startTime).Milliseconds(),
					userAgent,
				)
				return
			}))

	return &Server{
		Logg: logger,
		App:  app,

		HTTPHost:        httpHost,
		GRPCHost:        grpcHost,
		ShutdownTimeout: shutDownTimeout,

		grpcServer: grpcServer,
		httpServer: httpServer,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.Logg.Info().Msg("Starting server...")
	s.Logg.Info().Msgf("Start listening TCP connections on '%s' ...", s.GRPCHost)
	tcpListener, err := net.Listen("tcp", s.GRPCHost)
	if err != nil {
		s.Logg.Fatal().Err(err).Msgf("failed to start listen TCP connections on %s", s.GRPCHost)
	}
	s.tcpListener = tcpListener

	desc.RegisterEventServiceServer(s.grpcServer, s.App)

	httpServerErrChan := make(chan error)
	go func() {
		defer close(httpServerErrChan)
		s.Logg.Info().Msgf("Starting HTTP server on '%s' ...", s.HTTPHost)
		httpServerErrChan <- s.httpServer.ListenAndServe()
	}()

	grpcServerErrChan := make(chan error)
	go func() {
		defer close(grpcServerErrChan)
		s.Logg.Info().Msgf("Starting GRPC server on '%s' ...", s.GRPCHost)
		grpcServerErrChan <- s.grpcServer.Serve(tcpListener)
	}()

	select {
	case <-ctx.Done():
		s.Logg.Info().Msg("Stop server by signal")
	case httpServerErr := <-httpServerErrChan:
		if httpServerErr != nil {
			s.Logg.Error().Err(httpServerErr).Msg("failed to start http server")
			return httpServerErr
		}
		s.Logg.Info().Msg("HTTP server is stopped")
		return nil
	case grpcServerErr := <-grpcServerErrChan:
		if grpcServerErr != nil {
			s.Logg.Error().Err(grpcServerErr).Msg("failed to start grpc server")
			return grpcServerErr
		}
		s.Logg.Info().Msg("GRPC server is stopped")
		return nil
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) {
	s.Logg.Info().Msg("Start stopping server...")
	ctx, cancel := context.WithTimeout(ctx, s.ShutdownTimeout)
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.Logg.Error().Err(err).Msg("failed to shutdown http server")
			return
		}
		s.Logg.Info().Msg("Successfully shutdown HTTP server")
	}()
	go func() {
		defer wg.Done()
		// already have timeout - see NewServer method
		s.grpcServer.GracefulStop()
		s.Logg.Info().Msg("Successfully shutdown GRPC server")
	}()
	defer func() {
		defer wg.Done()
		if tcpListenerCloseErr := s.tcpListener.Close(); tcpListenerCloseErr != nil {
			s.Logg.Error().Err(tcpListenerCloseErr).Msg("Failed to close TCP listener")
		}
		s.Logg.Info().Msg("Successfully shutdown TCP listener")
	}()
	wg.Wait()
}
