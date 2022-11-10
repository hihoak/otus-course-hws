package internal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/api/exporter"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	desc "github.com/hihoak/otus-course-hws/sys-exporter/pkg/api/sys-exporter"
)

const (
	serverSnapshots  = "serverSnapshots"
	storageSnapshots = "storageSnapshots"
)

// Storager - load and saves all data what exporting during work.
type Storager interface {
	Save(ctx context.Context, data []byte, timestamp time.Time) error
}

// Collectorer - collecting (exporting) data from system, depending on arch.
type Collectorer interface {
	Export(ctx context.Context, timeNow time.Time) (*datastructures.SysData, error)
}

// Snapshoter - convert all metrics to a snapshot depending on configuration.
type Snapshoter interface {
	Push(ctx context.Context, data *datastructures.SysData)
	CreateSnapshots(ctx context.Context) <-chan *datastructures.SysData
	Close(ctx context.Context)
}

// Clockwork - clock interface.
type Clockwork interface {
	Now() time.Time
}

// Serverer - one who cares GRPC server.
type Serverer interface {
	Start(exporterService desc.ExporterServiceServer) error
	Stop()
}

type Implementation struct {
	cfg  *config.Config
	logg *logger.Logger

	doneChan chan interface{}

	storage   Storager
	collector Collectorer
	snapshots Snapshoter
	clock     Clockwork
	server    Serverer
}

func NewImpl(
	_ context.Context,
	cfg *config.Config,
	logg *logger.Logger,
	clock Clockwork,
	storage Storager,
	snapshots Snapshoter,
	colector Collectorer,
	server Serverer,
) *Implementation {
	return &Implementation{
		cfg: cfg,

		doneChan: make(chan interface{}),

		logg:      logg,
		storage:   storage,
		collector: colector,
		clock:     clock,
		snapshots: snapshots,
		server:    server,
	}
}

func (i *Implementation) startCollector(ctx context.Context, errChan chan<- error) <-chan *datastructures.SysData {
	i.logg.Info().Msg("starting collector...")
	scrapeTicker := time.NewTicker(i.cfg.Exporter.ScrapeInterval)

	dataChan := make(chan *datastructures.SysData, i.cfg.Exporter.DataChannelBuffer)
	go func() {
		defer scrapeTicker.Stop()
		defer close(dataChan)
		for {
			select {
			case <-scrapeTicker.C:
				go func() {
					i.logg.Debug().Msg("start exporting...")
					data, err := i.collector.Export(ctx, i.clock.Now())
					if err != nil {
						i.logg.Error().Err(err).Msg("failed to export data")
						errChan <- err
					}
					i.logg.Debug().Msg("successfully export info")
					dataChan <- data
				}()
			case <-i.doneChan:
				i.logg.Info().Msg("healthily stop exporting data. Got a stop signal")
				return
			}
		}
	}()
	return dataChan
}

func (i *Implementation) startStorage(ctx context.Context, data <-chan *datastructures.SysData, errChan chan<- error) {
	i.logg.Info().Msg("starting storage...")
	var end bool
	var timer *time.Timer
	for d := range data {
		select {
		case <-i.doneChan:
			if end {
				i.logg.Error().Msg("graceful shutdown time is over, maybe we doesn't store some data")
				return
			}
			if timer == nil {
				go func() {
					i.logg.Warn().
						Msgf("Storager: start timer of gracefully shutdown work of storage, "+
							"for waiting to store all snapshots '%s'",
							i.cfg.Exporter.GracefullyShutdownTimeout)
					timer = time.NewTimer(i.cfg.Exporter.GracefullyShutdownTimeout)
					<-timer.C
					end = true
				}()
			}
		default:
		}

		jsonData, marshallErr := json.Marshal(d)
		if marshallErr != nil {
			i.logg.Error().Err(marshallErr).Msg("failed to marshall data to json to store it in storage")
			continue
		}

		err := i.storage.Save(ctx, jsonData, d.TimeNow)
		if err != nil {
			i.logg.Error().Err(err).Msg("failed to store data")
			errChan <- err
			continue
		}
		i.logg.Debug().Msgf("successfully store data into")
	}
	i.logg.Info().Msg("Storager: Data channel is closed all data was saved. Healthily stopping storager...")
}

func (i *Implementation) startSnapshots(
	ctx context.Context,
	data <-chan *datastructures.SysData,
	_ chan<- error,
) <-chan *datastructures.SysData {
	i.logg.Info().Msg("starting snapshoter...")

	snapshotsChan := i.snapshots.CreateSnapshots(ctx)
	go func() {
		defer i.snapshots.Close(ctx)

		var end bool
		var timer *time.Timer
		for d := range data {
			select {
			case <-i.doneChan:
				if end {
					i.logg.Error().Msg("graceful shutdown time is over, maybe we doesn't convert some data to snapshots")
					return
				}
				if timer == nil {
					go func() {
						i.logg.Warn().
							Msgf("Snapshoter: start timer of gracefully shutdown work of snapshoter, "+
								"for proceed all metrics to snapshots '%s'",
								i.cfg.Exporter.GracefullyShutdownTimeout)
						timer = time.NewTimer(i.cfg.Exporter.GracefullyShutdownTimeout)
						<-timer.C
						end = true
					}()
				}
			default:
			}

			i.snapshots.Push(ctx, d)
			time.Sleep(time.Millisecond * 300)
			i.logg.Debug().Msg("successfully push data to snapshoter")
		}
		i.logg.Info().Msg("Data channel is closed all data was pushed to snapshoter. Healthily stopping snapshoter...")
	}()

	return snapshotsChan
}

func (i *Implementation) startServer(
	_ context.Context,
	data <-chan *datastructures.SysData,
	serverErrChan chan error,
) {
	i.logg.Info().Msg("Starting API server...")
	api := exporter.NewExporterService(i.logg, data)
	go func() {
		if err := i.server.Start(api); err != nil {
			serverErrChan <- err
			i.logg.Error().Err(err).Msg("grpc server returns error")
		}
	}()
	defer i.server.Stop()
	for range i.doneChan {
		i.logg.Info().Msg("got a stop signal, stopping API server")
	}
}

func (i *Implementation) broadcastChan(
	_ context.Context,
	data <-chan *datastructures.SysData,
) map[string]chan *datastructures.SysData {
	channels := map[string]chan *datastructures.SysData{
		storageSnapshots: make(chan *datastructures.SysData),
		serverSnapshots:  make(chan *datastructures.SysData),
	}

	go func() {
		for d := range data {
			for _, ch := range channels {
				ch <- d
			}
		}
	}()

	return channels
}

func (i *Implementation) Start(ctx context.Context) error {
	i.logg.Info().Msg("Start exporting data...")

	collectorErrChan := make(chan error)
	dataChan := i.startCollector(ctx, collectorErrChan)

	snapshotsErrChan := make(chan error)
	snapshotsChan := i.startSnapshots(ctx, dataChan, snapshotsErrChan)

	snapshotsChannels := i.broadcastChan(ctx, snapshotsChan)
	storageErrChan := make(chan error)
	go i.startStorage(ctx, snapshotsChannels[storageSnapshots], storageErrChan)
	serverErrChan := make(chan error)
	go i.startServer(ctx, snapshotsChannels[serverSnapshots], serverErrChan)

	go func() {
		for err := range storageErrChan {
			i.logg.Error().Err(err).Msg("storage error")
		}
	}()
	go func() {
		for err := range collectorErrChan {
			i.logg.Error().Err(err).Msg("collector error")
		}
	}()
	go func() {
		for err := range snapshotsErrChan {
			i.logg.Error().Err(err).Msg("snapshots error")
		}
	}()

	return <-serverErrChan
}

func (i *Implementation) Stop(ctx context.Context) error {
	i.logg.Info().Msg("Stopping exporting data...")
	close(i.doneChan)
	return nil
}
