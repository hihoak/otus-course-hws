package collector

import (
	"context"
	"sync"
	"time"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	collectorfuntions "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-funtions"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	"github.com/pkg/errors"
)

type Collector struct {
	logg *logger.Logger

	metricFunctions collectorfuntions.CollectFunctions
}

func New(cfg config.CollectorSection, logg *logger.Logger) *Collector {
	metricFunctions := make(collectorfuntions.CollectFunctions)
	if !cfg.DisableMetrics.LoadAverage {
		if f, ok := collectorfuntions.ExporterFunctions[collectorfuntions.LoadAverage]; !ok {
			logg.Warn().Msgf("Collector: load average metric is not supported yet")
		} else {
			logg.Debug().Msgf("Collector: will collect load average")
			metricFunctions[collectorfuntions.LoadAverage] = f
		}
	}
	if !cfg.DisableMetrics.CPUUsage {
		if f, ok := collectorfuntions.ExporterFunctions[collectorfuntions.CPUUsage]; !ok {
			logg.Warn().Msgf("Collector: cpu usage metric is not supported yet")
		} else {
			logg.Debug().Msgf("Collector: will collect cpu usage")
			metricFunctions[collectorfuntions.CPUUsage] = f
		}
	}
	if !cfg.DisableMetrics.DiskUsage {
		if f, ok := collectorfuntions.ExporterFunctions[collectorfuntions.DiskUsage]; !ok {
			logg.Warn().Msgf("Collector: disk usage metric is not supported yet")
		} else {
			logg.Debug().Msgf("Collector: will collect disk usage")
			metricFunctions[collectorfuntions.DiskUsage] = f
		}
	}
	if !cfg.DisableMetrics.NetworkTopTalkers {
		if f, ok := collectorfuntions.ExporterFunctions[collectorfuntions.NetworkTopTalkers]; !ok {
			logg.Warn().Msgf("Collector: network top talkers metric is not supported yet")
		} else {
			logg.Debug().Msgf("Collector: will collect network top talkers")
			metricFunctions[collectorfuntions.NetworkTopTalkers] = f
		}
	}
	if !cfg.DisableMetrics.FileSystemInfo {
		if f, ok := collectorfuntions.ExporterFunctions[collectorfuntions.FileSystemInfo]; !ok {
			logg.Warn().Msgf("Collector: file system info metric is not supported yet")
		} else {
			logg.Debug().Msgf("Collector: will collect file system info")
			metricFunctions[collectorfuntions.FileSystemInfo] = f
		}
	}

	return &Collector{
		logg: logg,

		metricFunctions: metricFunctions,
	}
}

func (c *Collector) Export(ctx context.Context, timeNow time.Time) (*datastructures.SysData, error) {
	c.logg.Debug().Msgf("start exporting data")
	data := &datastructures.SysData{
		TimeNow: timeNow,
	}

	multiError := collectorerrors.NewMultiError()
	wg := sync.WaitGroup{}
	mu := &sync.Mutex{}
	wg.Add(len(c.metricFunctions))
	for _, metricFunc := range c.metricFunctions {
		go func(f func(
			ctx context.Context,
			mu *sync.Mutex,
			logg *logger.Logger,
			data *datastructures.SysData,
		) *collectorerrors.ExportError,
		) {
			defer wg.Done()
			err := f(ctx, mu, c.logg, data)
			if err != nil {
				multiError.Append(err)
			}
		}(metricFunc)
	}
	wg.Wait()

	if multiError.Len() != 0 {
		return data, errors.Wrap(multiError, "failed to get full information. Some methods returns error")
	}
	c.logg.Debug().Msg("all information successfully exported!")
	return data, nil
}
