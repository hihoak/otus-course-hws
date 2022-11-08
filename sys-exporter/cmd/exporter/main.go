package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/server"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/clockwork"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/storage/memorystorage"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/amd64"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/snapshots"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

const (
	defaultConfigPath = ".exporter.yaml"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", defaultConfigPath, "path to config file")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	cfg := config.New(configPath)
	logg := logger.New(cfg.Logger)

	collector := amd64.New(logg)

	snapshoter := snapshots.New(ctx, logg, cfg.Snapshots)

	storager, err := memorystorage.New(cfg.MemoryStorage, logg)
	if err != nil {
		logg.Fatal().Err(err).Msg("failed to initialize storage")
	}

	clock := clockwork.New()

	serv := server.New(cfg.Server, logg)

	impl := internal.NewImpl(ctx, cfg, logg, clock, storager, snapshoter, collector, serv)
	if startErr := impl.Start(ctx); startErr != nil {
		logg.Fatal().Err(startErr).Msg("failed to start implementation")
	}
	defer func() {
		if err := impl.Stop(ctx); err != nil {
			logg.Error().Err(err).Msg("something goes wrong when stopping implementation")
		}
	}()
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
