package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/scheduler"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/sequencer/rabbit"
	memorystorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/sql"
)

var configFile string

const (
	rabbitClientName = "calendar-scheduler"
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/calendar/.calendar_config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.NewConfig(ctx, configFile)
	if err != nil {
		log.Fatal("failed to initialize config: ", err)
	}
	logg := logger.New(cfg.Logger.Level)
	logg.Info().Msg("Start initializing scheduler. Initialize connection to DB...")
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
	logg.Info().Msg("Successfully connected to DB. Start connection to sequence...")
	rabb := rabbit.NewCLient(logg, cfg.Rabbit.RabbitURL,
		rabbitClientName, cfg.Rabbit.ExchangesNames,
		cfg.Rabbit.QueueNames, cfg.Rabbit.Bindings)
	if rabbitErr := rabb.Connect(); rabbitErr != nil {
		logg.Fatal().Err(rabbitErr).Msg("failed connect to rabbitMQ...")
	}
	defer func() {
		if closeErr := rabb.Close(); closeErr != nil {
			logg.Error().Err(err).Msg("failed to close connection to rabbitMQ")
		}
	}()

	sch := scheduler.NewSchedulerImpl(logg, st,
		rabb, cfg.Scheduler.ScanPeriod,
		time.Hour*24*time.Duration(cfg.Scheduler.EventsDeprecationAgeInDays),
		cfg.Scheduler.NotifyPeriod, cfg.Scheduler.ExchangeToNotifyEvents)

	defer func() {
		if stopErr := sch.Stop(ctx); stopErr != nil {
			logg.Error().Err(stopErr).Msg("failed to stop scheduler")
		}
	}()
	if startErr := sch.Start(ctx); startErr != nil {
		logg.Error().Err(startErr).Msg("failed to start scheduler")
		return
	}
	logg.Info().Msg("successfully stop schedulers work...")
}
