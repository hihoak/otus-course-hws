//go:build macos
// +build macos

package collectorfuntions

import (
	"context"
	"fmt"
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
	topCpuRegexp         = regexp.MustCompile(`([0-9]+\.[0-9]+)% user, ([0-9]+\.[0-9]+)% sys, ([0-9]+\.[0-9]+)% idle`)
	networkTalkerRegexp  = regexp.MustCompile(`(.*)\.([0-9]+)\s+([0-9]+)\s+([0-9]+)`)
	fileSystemInfoRegexp = regexp.MustCompile(`(.*)\s+([0-9]+)\s+([0-9]+)\s+([0-9]+)\s+([0-9]+)%\s+([0-9]+)\s+([0-9]+)\s+([0-9]+)%\s+(.*)`)
)

var ExporterFunctions = CollectFunctions{
	LoadAverage:       getLoadAverage,
	CPUUsage:          getCpuUsage,
	DiskUsage:         diskUsage,
	NetworkTopTalkers: networkTopTalkers,
	FileSystemInfo:    fileSystemInfo,
}

func getLoadAverage(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting load average")
	res, runErr := execCMD("sysctl", "-n", "vm.loadavg")
	if runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to get load average: %v", runErr.Error()),
		}
	}
	// loadAvgCmd.String() - returns something like this { 1.78 1.94 2.08 }
	trimmedSpace := strings.TrimSpace(res)
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
	laFor5Min, err := strconv.ParseFloat(loadAverages[1], 32)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse float: %s: %v", loadAverages[1], err),
		}
	}
	laFor15min, err := strconv.ParseFloat(loadAverages[2], 32)
	if err != nil {
		return &collectorerrors.ExportError{
			FuncName: "load average",
			Reason:   fmt.Sprintf("failed to parse float: %s: %v", loadAverages[2], err),
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
	res, runErr := execCMD("top", "-l", "1")
	if runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "cpu percentage",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	var userCpu, sysCpu, idleCpu float64
	for _, line := range strings.Split(res, "\n") {
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
	mu.Unlock()
	logg.Debug().Msgf("successfully export CPU usage: %v", data.CPUUsage)
	return nil
}

func diskUsage(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting disk usage")
	res, runErr := execCMD("iostat", "-d")
	if runErr != nil {
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
	lines := strings.Split(res, "\n")
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
	mu.Unlock()
	logg.Debug().Msgf("successfully export Disk usage: %v", data.DiskUsage)
	return nil
}

func networkTopTalkers(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting network top talkers")
	res, runErr := execCMD("nettop", "-P", "-l", "1", "-J", "bytes_in,bytes_out", "-x")
	if runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "network top talkers",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	// 					bytes_in       bytes_out
	//some_talker.770 		   0           44670
	//some_talker2.775 		8905            9408
	networkTalkersStr := strings.Split(res, "\n")
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
	mu.Unlock()
	logg.Debug().Msgf("successfully export network talkers: %d", len(data.NetworkTalkers.ByBytesOut))
	return nil
}

func fileSystemInfo(
	_ context.Context,
	mu *sync.Mutex,
	logg *logger.Logger,
	data *datastructures.SysData,
) *collectorerrors.ExportError {
	logg.Debug().Msg("start getting fileSystem info")
	res, runErr := execCMD("df")
	if runErr != nil {
		return &collectorerrors.ExportError{
			FuncName: "file system info",
			Reason:   fmt.Sprintf("failed to run command: %v", runErr),
		}
	}
	// Filesystem                                            512-blocks      Used Available Capacity iused      ifree %iused  Mounted on
	// /dev/disk3s1s1                                         965595304  30093160 377103368     8%  502068 1885516840    0%   /
	// devfs                                                        423       423         0   100%     732          0  100%   /dev
	outputStrs := strings.Split(res, "\n")
	if len(outputStrs) < 1 {
		return &collectorerrors.ExportError{
			FuncName: "file system info",
			Reason:   "failed to parse command output expect at least one line",
		}
	}
	fileSystems := make([]datastructures.FileSystem, 0, len(outputStrs)-1)
	for _, line := range outputStrs[1:] {
		if line == "" {
			continue
		}
		stats := fileSystemInfoRegexp.FindStringSubmatch(strings.TrimSpace(line))
		if len(stats) != 10 {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse line expect 10 arguments got %d", len(stats)),
			}
		}
		filesystem := stats[1]

		blocksCount, err := strconv.Atoi(stats[2])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse blocks size to int: %v", err),
			}
		}
		bytesSize := blocksCount * 512

		blocksUsed, err := strconv.Atoi(stats[3])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse used blocks to int: %v", err),
			}
		}
		bytesUsed := blocksUsed * 512

		blockAvail, err := strconv.Atoi(stats[4])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse abailable blocks to int: %v", err),
			}
		}
		bytesAvalilable := blockAvail * 512

		var capacityPercent float32 = 100.0
		if blocksCount != 0 {
			capacityPercent = float32(blocksUsed) / float32(blocksCount) * 100
		}

		inodeUsed, err := strconv.Atoi(stats[6])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse used inodes to int: %v", err),
			}
		}

		inodeFree, err := strconv.Atoi(stats[7])
		if err != nil {
			return &collectorerrors.ExportError{
				FuncName: "file system info",
				Reason:   fmt.Sprintf("failed to parse used inodes to int: %v", err),
			}
		}

		var inodeUsedPercent float32 = 100.0
		if inodeUsed+inodeFree != 0 {
			inodeUsedPercent = float32(inodeUsed) / float32(inodeUsed+inodeFree) * 100
		}

		mountedOn := stats[9]

		fileSystems = append(fileSystems, datastructures.FileSystem{
			FileSystem: filesystem,
			MemoryInfo: datastructures.FileSystemMemoryInfo{
				SizeBytes:       bytesSize,
				UsedBytes:       bytesUsed,
				AvailableBytes:  bytesAvalilable,
				CapacityPercent: capacityPercent,
			},
			InodeInfo: datastructures.FileSystemInodeInfo{
				InodeFree:        inodeFree,
				InodeUsed:        inodeUsed,
				InodeUsedPercent: inodeUsedPercent,
			},
			MountedOn: mountedOn,
		})
	}

	sort.Slice(fileSystems, func(i, j int) bool {
		return fileSystems[i].MemoryInfo.SizeBytes > fileSystems[j].MemoryInfo.SizeBytes
	})

	mu.Lock()
	data.FileSystemInfo = &datastructures.FileSystemInfo{
		FileSystems: fileSystems,
	}
	mu.Unlock()
	logg.Debug().Msgf("successfully export filesystem info: %d", len(data.FileSystemInfo.FileSystems))
	return nil
}
