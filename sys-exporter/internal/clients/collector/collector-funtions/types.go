package collectorfuntions

import (
	"context"
	"sync"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

type metricFunctionsNames int

const (
	LoadAverage metricFunctionsNames = iota
	CPUUsage
	DiskUsage
	NetworkTopTalkers
	FileSystemInfo
)

type CollectFunctions map[metricFunctionsNames]func(
	ctx context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError
