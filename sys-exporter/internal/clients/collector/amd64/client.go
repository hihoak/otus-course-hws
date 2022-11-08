package amd64

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"

	"github.com/pkg/errors"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

type CollectorAMD64 struct {
	logg *logger.Logger

	metricFunctions []func(ctx context.Context, logg *logger.Logger, data *datastructures.SysData) *collectorerrors.ExportError
}

func New(logg *logger.Logger) *CollectorAMD64 {
	return &CollectorAMD64{
		logg: logg,

		metricFunctions: []func(ctx context.Context, logg *logger.Logger, data *datastructures.SysData) *collectorerrors.ExportError{getLoadAverage},
	}
}

func (c *CollectorAMD64) Export(ctx context.Context, timeNow time.Time) (*datastructures.SysData, error) {
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

func getLoadAverage(ctx context.Context, logg *logger.Logger, data *datastructures.SysData) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting load average")
	loadAvgCmd := exec.Command("sysctl", "-n", "vm.loadavg")
	out := bytes.Buffer{}
	loadAvgCmd.Stdout = &out
	runErr := loadAvgCmd.Run()
	if runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to get load average: %v", runErr.Error()),
		}
	}
	// loadAvgCmd.String() - returns something like this { 1.78 1.94 2.08 }
	trimmedSpace := strings.TrimSpace(out.String())
	removedBrackets := strings.Trim(trimmedSpace, "{}")
	trimmedSpace = strings.TrimSpace(removedBrackets)
	loadAverages := strings.Split(trimmedSpace, " ")
	if len(loadAverages) != 3 {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse load average output"),
		}
	}

	laFor1Min, err := strconv.ParseFloat(loadAverages[0], 64)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse float: %s: %v", loadAverages[0], err),
		}
	}
	laFor5Min, err := strconv.ParseFloat(loadAverages[1], 64)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse float: %s: %v", loadAverages[0], err),
		}
	}
	laFor15min, err := strconv.ParseFloat(loadAverages[2], 64)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse float: %s: %v", loadAverages[0], err),
		}
	}

	data.LoadAverage = &datastructures.LoadAverage{
		For1Min:  laFor1Min,
		For5min:  laFor5Min,
		For15min: laFor15min,
	}

	logg.Debug().Msgf("successfully got load average { %f %f %f }", laFor1Min, laFor5Min, laFor15min)
	return nil
}
