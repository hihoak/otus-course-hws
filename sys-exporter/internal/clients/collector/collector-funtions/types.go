package collectorfuntions

import (
	"context"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

type metricFunctionsNames int

const (
	LoadAverage metricFunctionsNames = iota
)

type CollectFunctions map[metricFunctionsNames]func(
	ctx context.Context,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError
