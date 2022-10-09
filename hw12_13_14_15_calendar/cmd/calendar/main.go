package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	internalhttp "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/server/http"
	memorystorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/sql"
	_ "github.com/lib/pq"
	"github.com/prometheus/common/log"
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

	server := internalhttp.NewServer(logg, calendar,
		config.Server.Host, config.Server.Port, config.Server.ReadTimeout,
		config.Server.WriteTimeout, config.Server.ShutDownTimeout)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	stopChan := make(chan interface{})
	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if stopError := server.Stop(ctx); stopError != nil {
			logg.Error().Err(stopError).Msg("failed to stop http server")
		}
		defer close(stopChan)
	}()

	logg.Info().Msg("calendar is running...")

	if err := server.Start(ctx); err != nil {
		logg.Error().Err(err).Msg("failed to start http server")
		cancel()
		select {
		// trying to gracefully shutdown
		case <-time.After(time.Second * 3):
			logg.Info().Msg("time of graceful shutdown is over :(")
		case <-stopChan:
			logg.Info().Msg("stopped until graceful shutdown is over")
		}
		os.Exit(1) //nolint:gocritic
	}
}
