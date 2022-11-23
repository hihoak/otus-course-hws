//go:build macos
// +build macos

package collectorfuntions

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	collectorerrors "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/collector/collector-errors"
	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
)

var (
	topCpuRegexp        = regexp.MustCompile(`([0-9]+\.[0-9]+)% user, ([0-9]+\.[0-9]+)% sys, ([0-9]+\.[0-9]+)% idle`)
	networkTalkerRegexp = regexp.MustCompile(`(.*)\.([0-9]+)\s+([0-9]+)\s+([0-9]+)`)
)

var ExporterFunctions = CollectFunctions{
	LoadAverage:       getLoadAverage,
	CPUUsage:          getCpuUsage,
	DiskUsage:         diskUsage,
	NetworkTopTalkers: networkTopTalkers,
}

func getLoadAverage(
	_ context.Context,
	mu *sync.Mutex,
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
	loadAverages := strings.Fields(trimmedSpace)
	if len(loadAverages) != 3 {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   "failed to parse load average output",
		}
	}

	laFor1Min, err := strconv.ParseFloat(loadAverages[0], 32)
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

	mu.Lock()
	data.LoadAverage = &datastructures.LoadAverage{
		For1Min:  float32(laFor1Min),
		For5min:  float32(laFor5Min),
		For15min: float32(laFor15min),
	}
	mu.Unlock()

	logg.Debug().Msgf("successfully got load average { %f %f %f }", laFor1Min, laFor5Min, laFor15min)
	return nil
}

func getCpuUsage(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting cpu usage percentage")
	topCmd := exec.Command("top", "-l", "1")
	out := bytes.Buffer{}
	topCmd.Stdout = &out
	if runErr := topCmd.Run(); runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "cpu percentage",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	var userCpu, sysCpu, idleCpu float64
	for _, line := range strings.Split(out.String(), "\n") {
		trimmedLine := strings.TrimSpace(line)
		// CPU usage: 4.32% user, 12.97% sys, 82.70% idle
		if !strings.HasPrefix(line, "CPU usage:") {
			continue
		}
		logg.Debug().Msgf("Collector: cpu percentage: got top output: %s", line)
		trimmedLine = strings.TrimSpace(strings.TrimLeft(trimmedLine, "CPU usage:"))
		cpuPercentages := topCpuRegexp.FindStringSubmatch(trimmedLine)
		logg.Debug().Msgf("Collector: cpu percentage: got cpuPercentages: %v", cpuPercentages)
		if len(cpuPercentages) != 4 {
			return &collectorerrors.ExportError{
				FuncName: "cpu percentage",
				Reason:   fmt.Sprintf("failed to parse command output, expect 3 cpu percentages, but got: %d", len(cpuPercentages)-1),
			}
		}
		var err error
		if userCpu, err = strconv.ParseFloat(cpuPercentages[1], 32); err != nil {
			return &collectorerrors.ExportError{
				FuncName: "cpu percentage",
				Reason:   fmt.Sprintf("failed to parse userCpu percentage to float: %v", err),
			}
		}
		if sysCpu, err = strconv.ParseFloat(cpuPercentages[2], 32); err != nil {
			return &collectorerrors.ExportError{
				FuncName: "cpu percentage",
				Reason:   fmt.Sprintf("failed to parse sysCpu percentage to float: %v", err),
			}
		}
		if idleCpu, err = strconv.ParseFloat(cpuPercentages[3], 32); err != nil {
			return &collectorerrors.ExportError{
				FuncName: "cpu percentage",
				Reason:   fmt.Sprintf("failed to parse idleCpu percentage to float: %v", err),
			}
		}
		break
	}

	mu.Lock()
	data.CPUUsage = &datastructures.CPUUsage{
		User: float32(userCpu),
		Sys:  float32(sysCpu),
		Idle: float32(idleCpu),
	}
	logg.Debug().Msgf("successfully export CPU usage: %v", data.CPUUsage)
	mu.Unlock()
	return nil
}

func diskUsage(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting disk usage")
	topCmd := exec.Command("iostat", "-d")
	out := bytes.Buffer{}
	topCmd.Stdout = &out
	if runErr := topCmd.Run(); runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	//               disk0
	//    KB/t  tps  MB/s
	//   21.94   70  1.50
	var kbPerTransfer, mbPerSecond float64
	var transfersPerSecond int
	lines := strings.Split(out.String(), "\n")
	if len(lines) != 4 {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to parse command output: expected 3 lines, got %d", len(lines)),
		}
	}
	stats := strings.Fields(lines[2])
	if len(stats) != 3 {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to parse command output: expected 3 stats, got %d", len(stats)),
		}
	}
	kbPerTransfer, err := strconv.ParseFloat(strings.TrimSpace(stats[0]), 32)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to parse kbPerTransfer to float: %v", err),
		}
	}
	transfersPerSecond, err = strconv.Atoi(strings.TrimSpace(stats[1]))
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to parse transfersPerSecond to int: %v", err),
		}
	}
	mbPerSecond, err = strconv.ParseFloat(strings.TrimSpace(stats[2]), 32)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "disk usage",
			Reason:   fmt.Sprintf("failed to parse mbPerSecond to float: %v", err),
		}
	}

	mu.Lock()
	data.DiskUsage = &datastructures.DiskUsage{
		KbPerTransfer:      float32(kbPerTransfer),
		TransfersPerSecond: transfersPerSecond,
		MbPerSecond:        float32(mbPerSecond),
	}
	logg.Debug().Msgf("successfully export Disk usage: %v", data.DiskUsage)
	mu.Unlock()
	return nil
}

func networkTopTalkers(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting network top talkers")
	topTalkerCmd := exec.Command("nettop", "-P", "-l", "1", "-J", "bytes_in,bytes_out", "-x")
	out := bytes.Buffer{}
	topTalkerCmd.Stdout = &out
	if runErr := topTalkerCmd.Run(); runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "network top talkers",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	// 					bytes_in       bytes_out
	//some_talker.770 		   0           44670
	//some_talker2.775 		8905            9408
	networkTalkersStr := strings.Split(out.String(), "\n")
	if len(networkTalkersStr) < 1 {
		return &collectorerrors.ExportError{
			FuncName: "network top talkers",
			Reason:   "failed to parse command output expect at least one line",
		}
	}
	networkTalkers := make([]*datastructures.NetworkTalker, 0, len(networkTalkersStr)-1)
	for _, line := range networkTalkersStr[1:] {
		if line == "" {
			continue
		}
		stats := networkTalkerRegexp.FindStringSubmatch(line)
		if len(stats) != 5 {
			return &collectorerrors.ExportError{
				FuncName: "network top talkers",
				Reason:   fmt.Sprintf("failed to parse line expect 4 arguments got %d", len(stats)),
			}
		}
		name := stats[1]
		if name == "" {
			name = "localhost"
		}
		pid, err := strconv.Atoi(stats[2])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "network top talkers",
				Reason:   fmt.Sprintf("failed to parse pid to int: %v", err),
			}
		}
		bytesIn, err := strconv.Atoi(stats[3])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "network top talkers",
				Reason:   fmt.Sprintf("failed to parse bytesIn stat to int: %v", err),
			}
		}
		bytesOut, err := strconv.Atoi(stats[4])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "network top talkers",
				Reason:   fmt.Sprintf("failed to parse bytesOut stat to int: %v", err),
			}
		}
		networkTalkers = append(networkTalkers, &datastructures.NetworkTalker{
			Name:     name,
			Pid:      pid,
			BytesIn:  bytesIn,
			BytesOut: bytesOut,
		})
	}

	byBytesIn := make([]*datastructures.NetworkTalker, len(networkTalkers))
	copy(byBytesIn, networkTalkers)
	sort.Slice(byBytesIn, func(i, j int) bool {
		return byBytesIn[i].BytesIn > byBytesIn[j].BytesIn
	})
	sort.Slice(networkTalkers, func(i, j int) bool {
		return networkTalkers[i].BytesOut > networkTalkers[j].BytesOut
	})

	mu.Lock()
	data.NetworkTalkers = &datastructures.NetworkTopTalkers{
		ByBytesIn:  byBytesIn,
		ByBytesOut: networkTalkers,
	}
	logg.Debug().Msgf("successfully export network talkers: %d", len(data.NetworkTalkers.ByBytesOut))
	mu.Unlock()
	return nil
}
