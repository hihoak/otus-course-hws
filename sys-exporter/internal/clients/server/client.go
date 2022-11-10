package server

import (
	"net"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	desc "github.com/hihoak/otus-course-hws/sys-exporter/pkg/api/sys-exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Server struct {
	logg *logger.Logger

	grpcServer *grpc.Server

	address string
}

func New(cfg config.ServerSection, logg *logger.Logger) *Server {
	return &Server{
		logg: logg,

		grpcServer: grpc.NewServer(),

		address: cfg.Address,
	}
}

func (s Server) Start(exporterService desc.ExporterServiceServer) error {
	s.logg.Info().Msg("starting grpc server...")
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return errors.Wrap(err, "failed to start listener on 'TCP' connections")
	}

	desc.RegisterExporterServiceServer(s.grpcServer, exporterService)

	s.logg.Info().Msg("Successfully register server. Now starting it...")
	return s.grpcServer.Serve(listener)
}

func (s Server) Stop() {
	s.logg.Info().Msg("Gracefully stopping GRPC server...")
	s.grpcServer.GracefulStop()
	s.logg.Info().Msg("GRPC server is stopped")
}
