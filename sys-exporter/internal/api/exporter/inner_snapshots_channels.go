package exporter

import (
	"sync"

	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
	"github.com/rs/xid"
)

type InnerSnapshots struct {
	buffer   int
	channels map[string]chan *datastructures.SysData

	mu *sync.Mutex
}

func NewInnerSnapshots(channelsBuffer int) *InnerSnapshots {
	return &InnerSnapshots{
		buffer:   channelsBuffer,
		channels: make(map[string]chan *datastructures.SysData),

		mu: &sync.Mutex{},
	}
}

func (i *InnerSnapshots) CreateNewChannel() (<-chan *datastructures.SysData, string) {
	newChan := make(chan *datastructures.SysData, i.buffer)
	i.mu.Lock()
	id := xid.New().String()
	i.channels[id] = newChan
	i.mu.Unlock()
	return newChan, id
}

func (i *InnerSnapshots) BroadcastSnapshot(snapshot *datastructures.SysData) {
	i.mu.Lock()
	for _, ch := range i.channels {
		ch <- snapshot
	}
	i.mu.Unlock()
}

func (i *InnerSnapshots) RemoveSnapshotChan(id string) {
	i.mu.Lock()
	close(i.channels[id])
	delete(i.channels, id)
	i.mu.Unlock()
}

func (i *InnerSnapshots) StopAll() {
	i.mu.Lock()
	for key, channel := range i.channels {
		close(channel)
		delete(i.channels, key)
	}
	i.mu.Unlock()
}
