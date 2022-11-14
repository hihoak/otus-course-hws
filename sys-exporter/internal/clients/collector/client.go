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
		logg.Debug().Msgf("Collector: will collect load average")
		metricFunctions[collectorfuntions.LoadAverage] = collectorfuntions.ExporterFunctions[collectorfuntions.LoadAverage]
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

	var multiError collectorerrors.MultiError
	wg := sync.WaitGroup{}
	wg.Add(len(c.metricFunctions))
	for _, metricFunc := range c.metricFunctions {
		go func(f func(ctx context.Context, logg *logger.Logger, data *datastructures.SysData) *collectorerrors.ExportError) {
			defer wg.Done()
			err := f(ctx, c.logg, data)
			if err != nil {
				multiError.Append(err)
			}
		}(metricFunc)
	}
	wg.Wait()

	if multiError.Error() != "" {
		return data, errors.Wrap(multiError, "failed to get full information. Some methods returns error")
	}
	c.logg.Debug().Msg("all information successfully exported!")
	return data, nil
}
