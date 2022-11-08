package exporter

import (
	"sync"
	"time"

	datastructures "github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/data-structures"
)

type InnerSnapshots struct {
	buffer   int
	channels map[int64]chan *datastructures.SysData

	mu *sync.Mutex
}

func NewInnerSnapshots(channelsBuffer int) *InnerSnapshots {
	return &InnerSnapshots{
		buffer:   channelsBuffer,
		channels: make(map[int64]chan *datastructures.SysData),

		mu: &sync.Mutex{},
	}
}

func (i *InnerSnapshots) CreateNewChannel() (<-chan *datastructures.SysData, int64) {
	newChan := make(chan *datastructures.SysData, i.buffer)
	i.mu.Lock()
	id := time.Now().UnixNano()
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

func (i *InnerSnapshots) RemoveSnapshotChan(id int64) {
	i.mu.Lock()
	close(i.channels[id])
	delete(i.channels, id)
	i.mu.Unlock()
}

func (i *InnerSnapshots) StopAll() {
	i.mu.Lock()
	keys := make([]int64, 0)
	for id := range i.channels {
		keys = append(keys, id)
	}
	for _, key := range keys {
		close(i.channels[key])
		delete(i.channels, key)
	}
	i.mu.Unlock()
}
