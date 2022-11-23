package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/clockwork"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/filesystem"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/server"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/snapshots"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/storage/diskstorage"
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
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	cfg := config.New(configPath)
	logg := logger.New(cfg.Logger)

	collectorer := collector.New(cfg.Collector, logg)

	snapshoter := snapshots.New(ctx, logg, cfg.Snapshots)

	fileSystem := filesystem.New()

	storager, err := diskstorage.New(cfg.DiskStorage, logg, fileSystem)
	if err != nil {
		logg.Fatal().Err(err).Msg("failed to initialize storage")
	}

	clock := clockwork.New()

	serv := server.New(cfg.Server, logg)

	impl := internal.NewImpl(ctx, cfg, logg, clock, storager, snapshoter, collectorer, serv)
	go func() {
		if startErr := impl.Start(ctx); startErr != nil {
			logg.Fatal().Err(startErr).Msg("failed to start implementation")
		}
	}()

	<-ctx.Done()
	cancel()

	logg.Info().Msg("Stopping exporting data...")

	time.Sleep(cfg.Exporter.GracefullyShutdownTimeout)
}
