package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/server"
	memorystorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/sql"
	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	version    bool
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/.calendar_config.yaml", "Path to configuration file")
	flag.BoolVar(&version, "version", false, "prints version")
}

func main() {
	flag.Parse()

	if version {
		printVersion()
		return
	}

	ctx := context.Background()

	cfg, err := config.NewConfig(ctx, configFile)
	if err != nil {
		log.Fatal("can't initialize config:", err)
	}
	logg := logger.New(cfg.Logger.Level)
	logg.Info().Msg("Successfully initialize config...")

	var st app.Storage
	if cfg.UseInMemoryStorage {
		st = memorystorage.New(logg)
	} else {
		sqlSt := sqlstorage.New(
			logg, cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
			cfg.Database.Password, cfg.Database.DBName, cfg.Database.ConnectionTimeout,
			cfg.Database.OperationTimeout)
		if connectionErr := sqlSt.Connect(ctx); connectionErr != nil {
			logg.Fatal().Err(connectionErr).Msg("failed to connect to database")
		}
		defer func() {
			if closeErr := sqlSt.Close(ctx); closeErr != nil {
				logg.Error().Err(closeErr).Msg("failed to close connection to database")
			}
		}()
		st = sqlSt
	}

	calendar := app.New(logg, st)

	mux, muxErr := GetGrpcGatewayMultiplexer(ctx, net.JoinHostPort(cfg.Server.Host, cfg.Server.GRPCPort))
	if muxErr != nil {
		logg.Fatal().Err(err).Msg("failed to register grpc-gateway Multiplexer")
	}

	calendarServer := server.NewServer(ctx, logg, calendar, mux,
		net.JoinHostPort(cfg.Server.Host, cfg.Server.HTTPPort),
		net.JoinHostPort(cfg.Server.Host, cfg.Server.GRPCPort), cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout, cfg.Server.ShutDownTimeout)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	stopChan := make(chan interface{})
	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		calendarServer.Stop(ctx)
		defer close(stopChan)
	}()

	if err := calendarServer.Start(ctx); err != nil {
		logg.Error().Err(err).Msg("failed to start server")
	}
	logg.Info().Msg("calendar is running...")
	select {
	// trying to gracefully shutdown
	case <-time.After(cfg.Server.GracefulShutdown):
		logg.Info().Msg("time of graceful shutdown is over :(")
	case <-stopChan:
		logg.Info().Msg("stopped until graceful shutdown is over")
	}
	logg.Info().Msg("calendar is stopped")
}

func GetGrpcGatewayMultiplexer(ctx context.Context, grpcHost string) (http.Handler, error) {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := desc.RegisterEventServiceHandlerFromEndpoint(ctx, mux, grpcHost, opts)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to register grpc-gateway on host %s", grpcHost))
	}

	return mux, nil
}
