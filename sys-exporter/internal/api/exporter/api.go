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
	doneChan := make(chan interface{})
	innerSnapshots := NewInnerSnapshots(cap(snapshots))
	go func() {
		for data := range snapshots {
			innerSnapshots.BroadcastSnapshot(data)
		}
		close(doneChan)
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
	e.logg.Debug().Msgf("SendStreamSnapshots: successfully created a new channel with id '%d'", id)
	defer e.innerSnapshotsChannels.RemoveSnapshotChan(id)
	for data := range ch {
		e.logg.Debug().Msgf("SendStreamSnapshots: start sending snapshot to channle with id '%d'", id)
		if err := stream.Send(&desc.SendStreamSnapshotsResponse{
			Snapshot: fromSnapshotToPb(data),
		}); err != nil {
			e.logg.Debug().
				Msgf("SendStreamSnapshots: stop sending snapshots to channel with id '%d'"+
					"because of fail to send to client", id)
			return errors.Wrap(err, "failed to send snapshot to client")
		}
	}
	e.logg.Debug().
		Msgf("SendStreamSnapshots: stop sending snapshots to channel with id '%d' because of closed channel", id)
	return io.EOF
}

func fromSnapshotToPb(data *datastructures.SysData) *desc.Snapshot {
	return &desc.Snapshot{
		Timestamp:   data.TimeNow.UnixNano(),
		LoadAverage: fromLoadAverageToPb(data.LoadAverage),
	}
}

func fromLoadAverageToPb(data *datastructures.LoadAverage) *desc.Snapshot_LoadAverage {
	return &desc.Snapshot_LoadAverage{
		For1Min:  float32(data.For1Min),
		For5Min:  float32(data.For5min),
		For15Min: float32(data.For15min),
	}
}
