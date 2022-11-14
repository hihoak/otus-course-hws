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
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/server"
	memorystorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/sql"
	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	_ "github.com/lib/pq"
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

	config, err := NewConfig(ctx, configFile)
	if err != nil {
		log.Fatal("can't initialize config:", err)
	}
	logg := logger.New(config.Logger.Level)
	logg.Info().Msg("Successfully initialize config...")

	var st app.Storage
	if config.UseInMemoryStorage {
		st = memorystorage.New(logg)
	} else {
		sqlSt := sqlstorage.New(
			logg, config.Database.Host, config.Database.Port, config.Database.User,
			config.Database.Password, config.Database.DBName, config.Database.ConnectionTimeout,
			config.Database.OperationTimeout)
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

	mux, muxErr := GetGrpcGatewayMultiplexer(ctx, net.JoinHostPort(config.Server.Host, config.Server.GRPCPort))
	if muxErr != nil {
		logg.Fatal().Err(err).Msg("failed to register grpc-gateway Multiplexer")
	}

	calendarServer := server.NewServer(ctx, logg, calendar, mux,
		net.JoinHostPort(config.Server.Host, config.Server.HTTPPort),
		net.JoinHostPort(config.Server.Host, config.Server.GRPCPort), config.Server.ReadTimeout,
		config.Server.WriteTimeout, config.Server.ShutDownTimeout)

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
	case <-time.After(time.Second * 10):
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
		return nil, fmt.Errorf("failed to register grpc-gateway on host %s: %w", grpcHost, err)
	}

	return mux, nil
}
