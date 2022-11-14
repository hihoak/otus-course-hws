//go:build ubuntu_amd_64
// +build ubuntu_amd_64

package collectorfuntions

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

var ExporterFunctions = CollectFunctions{
	LoadAverage: getLoadAverage,
}

func getLoadAverage(
	_ context.Context,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting load average")
	loadAvgCmd := exec.Command("cat", "/proc/loadavg")
	out := bytes.Buffer{}
	loadAvgCmd.Stdout = &out
	if runErr := loadAvgCmd.Run(); runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to get load average: %v", runErr.Error()),
		}
	}
	// out.String() - returns something like this: 0.07 0.02 0.00 1/510 98
	trimmedSpace := strings.TrimSpace(out.String())
	fmt.Println(trimmedSpace)
	loadAverageInfo := strings.Split(trimmedSpace, " ")
	if len(loadAverageInfo) < 3 {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   "parse error: unexpected number of arguments",
		}
	}

	loadAverages := make([]float64, 3)

	for idx, stringLA := range loadAverageInfo[:3] {
		floatLA, err := strconv.ParseFloat(stringLA, 64)
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "load average",
				Reason:   fmt.Sprintf("parse error: failed to convert la to float: %s", err),
			}
		}
		loadAverages[idx] = floatLA
	}

	data.LoadAverage = &datastructures.LoadAverage{
		For1Min:  loadAverages[0],
		For5min:  loadAverages[1],
		For15min: loadAverages[2],
	}

	logg.Debug().
		Msgf("successfully got load average { %f %f %f }",
			data.LoadAverage.For1Min, data.LoadAverage.For5min, data.LoadAverage.For15min)
	return nil
}
