package exporter

import (
	"io"

	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	desc "github.com/hihoak/otus-course-hws/sys-exporter/pkg/api/sys-exporter"
	"github.com/pkg/errors"
)

// ServiceAPI - API of exporter.
type ServiceAPI struct {
	desc.ExporterServiceServer
	logg *logger.Logger

	snapshots              <-chan *datastructures.SysData
	innerSnapshotsChannels *InnerSnapshots
}

func NewExporterService(logg *logger.Logger, snapshots <-chan *datastructures.SysData) *ServiceAPI {
	innerSnapshots := NewInnerSnapshots(cap(snapshots))
	go func() {
		for data := range snapshots {
			innerSnapshots.BroadcastSnapshot(data)
		}
		innerSnapshots.StopAll()
	}()
	return &ServiceAPI{
		logg: logg,

		snapshots:              snapshots,
		innerSnapshotsChannels: innerSnapshots,
	}
}

func (e *ServiceAPI) SendStreamSnapshots(
	_ *desc.SendStreamSnapshotsRequest,
	stream desc.ExporterService_SendStreamSnapshotsServer,
) error {
	e.logg.Debug().Msg("SendStreamSnapshots: got a connection for pulling snapshots")
	ch, id := e.innerSnapshotsChannels.CreateNewChannel()
	e.logg.Debug().Msgf("SendStreamSnapshots: successfully created a new channel with id '%s'", id)
	defer e.innerSnapshotsChannels.RemoveSnapshotChan(id)
	for data := range ch {
		e.logg.Debug().Msgf("SendStreamSnapshots: start sending snapshot to channel with id '%s'", id)
		if err := stream.Send(&desc.SendStreamSnapshotsResponse{
			Snapshot: fromSnapshotToPb(data),
		}); err != nil {
			e.logg.Debug().
				Msgf("SendStreamSnapshots: stop sending snapshots to channel with id '%s'"+
					"because of fail to send to client", id)
			return errors.Wrap(err, "failed to send snapshot to client")
		}
	}
	e.logg.Debug().
		Msgf("SendStreamSnapshots: stop sending snapshots to channel with id '%s' because of closed channel", id)
	return io.EOF
}

func fromSnapshotToPb(data *datastructures.SysData) *desc.Snapshot {
	if data == nil {
		return nil
	}
	return &desc.Snapshot{
		Timestamp:         data.TimeNow.UnixNano(),
		LoadAverage:       fromLoadAverageToPb(data.LoadAverage),
		CpuUsage:          fromCPUUsageToPb(data.CPUUsage),
		DiskUsage:         fromDiskUsageToPb(data.DiskUsage),
		NetworkTopTalkers: fromNetworkTopTalkersToPb(data.NetworkTalkers),
		FileSystemInfo:    fromFileSystemInfoToPb(data.FileSystemInfo),
	}
}

func fromLoadAverageToPb(data *datastructures.LoadAverage) *desc.Snapshot_LoadAverage {
	if data == nil {
		return nil
	}
	return &desc.Snapshot_LoadAverage{
		For1Min:     data.For1Min,
		For5Min:     data.For5min,
		For15Min:    data.For15min,
		CpuUsageWin: data.CPUPercentUsage,
	}
}

func fromCPUUsageToPb(data *datastructures.CPUUsage) *desc.Snapshot_CpuUsage {
	if data == nil {
		return nil
	}
	return &desc.Snapshot_CpuUsage{
		User: data.User,
		Sys:  data.Sys,
		Idle: data.Idle,
	}
}

func fromDiskUsageToPb(data *datastructures.DiskUsage) *desc.Snapshot_DiskUsage {
	if data == nil {
		return nil
	}
	return &desc.Snapshot_DiskUsage{
		KbPerTransfer:      data.KbPerTransfer,
		MbPerSecond:        data.MbPerSecond,
		TransfersPerSecond: int64(data.TransfersPerSecond),
	}
}

func fromNetworkTopTalkersToPb(data *datastructures.NetworkTopTalkers) *desc.Snapshot_NetworkTopTalkers {
	if data == nil {
		return nil
	}
	return &desc.Snapshot_NetworkTopTalkers{
		ByBytesIn:  fromNetworkTalkersToPbs(data.ByBytesIn),
		ByBytesOut: fromNetworkTalkersToPbs(data.ByBytesOut),
	}
}

func fromNetworkTalkersToPbs(data []*datastructures.NetworkTalker) []*desc.Snapshot_NetworkTopTalkers_NetworkTalker {
	res := make([]*desc.Snapshot_NetworkTopTalkers_NetworkTalker, len(data))
	for idx, talker := range data {
		res[idx] = &desc.Snapshot_NetworkTopTalkers_NetworkTalker{
			Name:     talker.Name,
			Pid:      int64(talker.Pid),
			BytesOut: int64(talker.BytesOut),
			BytesIn:  int64(talker.BytesIn),
		}
	}
	return res
}

func fromFileSystemInfoToPb(
	data *datastructures.FileSystemInfo,
) *desc.Snapshot_FileSystemInfo {
	if data == nil {
		return nil
	}
	return &desc.Snapshot_FileSystemInfo{
		FileSystem: fromFileSystemsToPb(data.FileSystems),
	}
}

func fromFileSystemsToPb(
	data []datastructures.FileSystem,
) []*desc.Snapshot_FileSystemInfo_FileSystem {
	res := make([]*desc.Snapshot_FileSystemInfo_FileSystem, len(data))
	for idx, d := range data {
		res[idx] = &desc.Snapshot_FileSystemInfo_FileSystem{
			FileSystem: d.FileSystem,
			MemoryInfo: fromFileSystemMemoryInfoToPb(d.MemoryInfo),
			InodeInfo:  fromFileSystemInodeInfoToPb(d.InodeInfo),
			MountedOn:  d.MountedOn,
		}
	}
	return res
}

func fromFileSystemMemoryInfoToPb(
	data datastructures.FileSystemMemoryInfo,
) *desc.Snapshot_FileSystemInfo_FileSystem_FileSystemMemoryInfo {
	return &desc.Snapshot_FileSystemInfo_FileSystem_FileSystemMemoryInfo{
		SizeBytes:       int64(data.SizeBytes),
		UsedBytes:       int64(data.UsedBytes),
		AvailableBytes:  int64(data.AvailableBytes),
		CapacityPercent: data.CapacityPercent,
	}
}

func fromFileSystemInodeInfoToPb(
	data datastructures.FileSystemInodeInfo,
) *desc.Snapshot_FileSystemInfo_FileSystem_FileSystemInodeInfo {
	return &desc.Snapshot_FileSystemInfo_FileSystem_FileSystemInodeInfo{
		InodeUsedPercent: data.InodeUsedPercent,
		InodeUsed:        int64(data.InodeUsed),
		InodeFree:        int64(data.InodeFree),
	}
}
