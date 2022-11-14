//go:build macos_arm_64
// +build macos_arm_64

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
	loadAvgCmd := exec.Command("sysctl", "-n", "vm.loadavg")
	out := bytes.Buffer{}
	loadAvgCmd.Stdout = &out
	if runErr := loadAvgCmd.Run(); runErr != nil {
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
			Reason:   "failed to parse load average output",
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
