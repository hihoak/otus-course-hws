package datastructures

import "time"

type LoadAverage struct {
	For1Min  float64
	For5min  float64
	For15min float64
}

type SysData struct {
	TimeNow     time.Time
	LoadAverage *LoadAverage
}
