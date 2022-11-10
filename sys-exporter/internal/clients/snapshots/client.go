package snapshots

import (
	"context"
	"fmt"
	"time"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

type Snapshots struct {
	logg *logger.Logger

	snapshotsChan chan *datastructures.SysData

	// main parameter of snapshots. Accumulates inside all data and after snapshotInterval it returns avg data
	totalSysData   *datastructures.SysData
	countOfSysData int64

	// how often to send a snapshots
	ticker *time.Ticker

	doneChan chan interface{}

	// config parameters
	// initial time on start of exporter when it's collecting metrics for first snapshot
	warmupInterval time.Duration
	// interval of sending snapshots
	snapshotInterval time.Duration

	itsWarmup bool
}

func New(ctx context.Context, logg *logger.Logger, cfg config.SnapshotsSection) *Snapshots {
	return &Snapshots{
		logg: logg,

		snapshotsChan: make(chan *datastructures.SysData),

		totalSysData: &datastructures.SysData{
			LoadAverage: &datastructures.LoadAverage{},
		},
		countOfSysData: 0,

		doneChan: make(chan interface{}),

		warmupInterval:   cfg.WarmupInterval,
		snapshotInterval: cfg.SnapshotInterval,

		itsWarmup: true,
	}
}

func (s *Snapshots) Push(ctx context.Context, data *datastructures.SysData) {
	s.logg.Debug().Msgf("got a new data after push: %v", *data)
	s.totalSysData.LoadAverage.For1Min += data.LoadAverage.For1Min
	s.totalSysData.LoadAverage.For5min += data.LoadAverage.For5min
	s.totalSysData.LoadAverage.For15min += data.LoadAverage.For15min
	s.countOfSysData++
}

func (s *Snapshots) CreateSnapshots(ctx context.Context) <-chan *datastructures.SysData {
	s.logg.Debug().Msg("start creating snapshots...")
	s.ticker = time.NewTicker(s.warmupInterval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.logg.Debug().Msg("start calculating snapshot")
				snapshot, calcErr := s.calculateSnapshot()
				if calcErr != nil {
					s.logg.Error().Err(calcErr).Msg("Snapshoter: failed to calculate snapshot")
					continue
				}
				s.snapshotsChan <- snapshot
				if s.itsWarmup {
					s.logg.Debug().Msgf("warmup is over now calculating snapshots with interval of '%s'", s.snapshotInterval)
					s.itsWarmup = false
					s.ticker.Reset(s.snapshotInterval)
				}
			case <-s.doneChan:
				close(s.snapshotsChan)
				s.logg.Info().Msg("creating snapshots is stopped")
				return
			}
		}
	}()
	return s.snapshotsChan
}

func (s *Snapshots) calculateSnapshot() (*datastructures.SysData, error) {
	if s.countOfSysData == 0 {
		return nil, fmt.Errorf("can't calculate snapshot, " +
			"because of zero metrics was received for the last snapshot interval")
	}
	snapshot := &datastructures.SysData{
		TimeNow: time.Now(),
		LoadAverage: &datastructures.LoadAverage{
			For1Min:  s.totalSysData.LoadAverage.For1Min / float64(s.countOfSysData),
			For5min:  s.totalSysData.LoadAverage.For5min / float64(s.countOfSysData),
			For15min: s.totalSysData.LoadAverage.For15min / float64(s.countOfSysData),
		},
	}
	s.totalSysData = &datastructures.SysData{
		LoadAverage: &datastructures.LoadAverage{},
	}
	s.countOfSysData = 0
	return snapshot, nil
}

func (s *Snapshots) Close(ctx context.Context) {
	s.logg.Info().Msg("start ending create snapshots")
	close(s.doneChan)
}
