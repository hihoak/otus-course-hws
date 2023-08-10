package snapshots

import (
	"context"
	"fmt"
	"sync"
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

	doneChan <-chan struct{}

	// config parameters
	// initial time on start of exporter when it's collecting metrics for first snapshot
	warmupInterval time.Duration
	// interval of sending snapshots
	snapshotInterval time.Duration

	itsWarmup bool

	mu *sync.Mutex
}

func New(ctx context.Context, logg *logger.Logger, cfg config.SnapshotsSection) *Snapshots {
	return &Snapshots{
		logg: logg,

		snapshotsChan: make(chan *datastructures.SysData),

		totalSysData: &datastructures.SysData{
			LoadAverage:    &datastructures.LoadAverage{},
			CPUUsage:       &datastructures.CPUUsage{},
			DiskUsage:      &datastructures.DiskUsage{},
			NetworkTalkers: &datastructures.NetworkTopTalkers{},
			FileSystemInfo: &datastructures.FileSystemInfo{},
		},
		countOfSysData: 0,

		doneChan: ctx.Done(),

		warmupInterval:   cfg.WarmupInterval,
		snapshotInterval: cfg.SnapshotInterval,

		itsWarmup: true,

		mu: &sync.Mutex{},
	}
}

func (s *Snapshots) Push(ctx context.Context, data *datastructures.SysData) {
	s.mu.Lock()
	s.logg.Debug().Msgf("got a new data after push: %v", *data)
	if data.LoadAverage != nil {
		s.totalSysData.LoadAverage.For1Min += data.LoadAverage.For1Min
		s.totalSysData.LoadAverage.For5min += data.LoadAverage.For5min
		s.totalSysData.LoadAverage.For15min += data.LoadAverage.For15min
		s.totalSysData.LoadAverage.CPUPercentUsage += data.LoadAverage.CPUPercentUsage
	}
	if data.CPUUsage != nil {
		s.totalSysData.CPUUsage.Sys += data.CPUUsage.Sys
		s.totalSysData.CPUUsage.User += data.CPUUsage.User
		s.totalSysData.CPUUsage.Idle += data.CPUUsage.Idle
	}
	if data.DiskUsage != nil {
		s.totalSysData.DiskUsage = data.DiskUsage
	}
	if data.NetworkTalkers != nil {
		s.totalSysData.NetworkTalkers = data.NetworkTalkers
	}
	if data.FileSystemInfo != nil {
		s.totalSysData.FileSystemInfo = data.FileSystemInfo
	}
	s.countOfSysData++
	s.mu.Unlock()
}

func (s *Snapshots) CreateSnapshots(_ context.Context) <-chan *datastructures.SysData {
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
			For1Min:         s.totalSysData.LoadAverage.For1Min / float32(s.countOfSysData),
			For5min:         s.totalSysData.LoadAverage.For5min / float32(s.countOfSysData),
			For15min:        s.totalSysData.LoadAverage.For15min / float32(s.countOfSysData),
			CPUPercentUsage: s.totalSysData.LoadAverage.CPUPercentUsage / float32(s.countOfSysData),
		},
		CPUUsage: &datastructures.CPUUsage{
			User: s.totalSysData.CPUUsage.User / float32(s.countOfSysData),
			Sys:  s.totalSysData.CPUUsage.Sys / float32(s.countOfSysData),
			Idle: s.totalSysData.CPUUsage.Sys / float32(s.countOfSysData),
		},
		DiskUsage:      s.totalSysData.DiskUsage,
		NetworkTalkers: s.totalSysData.NetworkTalkers,
		FileSystemInfo: s.totalSysData.FileSystemInfo,
	}
	s.totalSysData = &datastructures.SysData{
		LoadAverage:    &datastructures.LoadAverage{},
		CPUUsage:       &datastructures.CPUUsage{},
		DiskUsage:      &datastructures.DiskUsage{},
		NetworkTalkers: &datastructures.NetworkTopTalkers{},
	}
	s.countOfSysData = 0
	return snapshot, nil
}

func (s *Snapshots) Close(_ context.Context) {
	s.logg.Info().Msg("start ending create snapshots")
}
