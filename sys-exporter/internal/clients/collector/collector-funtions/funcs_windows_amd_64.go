//go:build windows_amd_64
// +build windows_amd_64

package collectorfuntions

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

var ExporterFunctions = CollectFunctions{
	LoadAverage: getLoadAverage,
}

func getLoadAverage(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting load average")
	loadAvgCmd := exec.Command("wmic", "cpu", "get", "loadpercentage")
	out := bytes.Buffer{}
	loadAvgCmd.Stdout = &out
	if runErr := loadAvgCmd.Run(); runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to get load average: %v", runErr.Error()),
		}
	}
	// loadAvgCmd.String() - returns something like this:
	// LoadPercentage
	// 1
	//
	//
	trimmedSpace := strings.TrimSpace(out.String())
	splittedByLines := strings.Split(trimmedSpace, "\n")
	if len(splittedByLines) != 2 {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   "failed to parse load average output",
		}
	}

	percentageCpuLoad, err := strconv.ParseFloat(splittedByLines[1], 32)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse load average to percents: %s", err),
		}
	}

	mu.Lock()
	data.LoadAverage = &datastructures.LoadAverage{
		CPUPercentUsage: float32(percentageCpuLoad),
	}
	mu.Unlock()

	logg.Debug().Msgf("successfully got cpu percent usage: %f", data.LoadAverage.CPUPercentUsage)
	return nil
}
