package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/sender"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/sequencer/rabbit"
	memorystorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/memory"
	sqlstorage "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage/sql"
	_ "github.com/lib/pq"
)

const (
	rabbitClientName = "calendar-sender"
)

var configFile string

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
	logg.Info().Msg("Start initializing scheduler. Initialize connection to rabbit...")
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

	logg.Info().Msg("Initialize connection to DB...")
	var st sender.Storager
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

	splitPath := strings.Split(cfg.Sender.OutputFile, "/")
	fmt.Println("creating directory in ", "/"+strings.Join(splitPath[:len(splitPath)-1], "/"))
	if err := os.MkdirAll(strings.Join(splitPath[:len(splitPath)-1], "/"), 0o777); err != nil {
		logg.Fatal().Err(err).Msg("failed to create dir")
	}
	outputFile, err := os.OpenFile(cfg.Sender.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o777)
	if err != nil {
		logg.Fatal().Err(err).Msgf("failed to open output file '%s'", outputFile.Name())
	}

	sndr := sender.NewSender(ctx, logg, rabb, st, outputFile, cfg.Sender.QueueToPullNotifications)

	defer func() {
		if stopErr := sndr.Stop(ctx); stopErr != nil {
			logg.Error().Err(stopErr).Msg("failed to stop sender")
		}
	}()
	if startErr := sndr.Start(ctx); startErr != nil {
		logg.Error().Err(startErr).Msg("failed to start sender")
		return
	}
	logg.Info().Msg("successfully stop senders work...")
}
