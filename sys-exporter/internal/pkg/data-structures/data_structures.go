package datastructures

import (
	"time"
)

type LoadAverage struct {
	CPUPercentUsage float32
	For1Min         float32
	For5min         float32
	For15min        float32
}

type CPUUsage struct {
	User float32
	Sys  float32
	Idle float32
}

type DiskUsage struct {
	KbPerTransfer      float32
	MbPerSecond        float32
	TransfersPerSecond int
}

type NetworkTalker struct {
	Name     string
	Pid      int
	BytesIn  int
	BytesOut int
}

type NetworkTopTalkers struct {
	ByBytesIn  []*NetworkTalker
	ByBytesOut []*NetworkTalker
}

type FileSystemMemoryInfo struct {
	SizeBytes       int
	UsedBytes       int
	AvailableBytes  int
	CapacityPercent float32
}

type FileSystemInodeInfo struct {
	InodeUsed        int
	InodeFree        int
	InodeUsedPercent float32
}

type FileSystem struct {
	FileSystem string
	MemoryInfo FileSystemMemoryInfo
	InodeInfo  FileSystemInodeInfo
	MountedOn  string
}

type FileSystemInfo struct {
	FileSystems []FileSystem
}

type SysData struct {
	TimeNow        time.Time
	LoadAverage    *LoadAverage
	CPUUsage       *CPUUsage
	DiskUsage      *DiskUsage
	NetworkTalkers *NetworkTopTalkers
	FileSystemInfo *FileSystemInfo
}
