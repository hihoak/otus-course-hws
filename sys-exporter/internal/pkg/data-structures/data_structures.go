package datastructures

import "time"

type LoadAverage struct {
	CPUPercentUsage float32
	For1Min         float32
	For5min         float32
	For15min        float32
}

type SysData struct {
	TimeNow     time.Time
	LoadAverage *LoadAverage
}
