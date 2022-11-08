package clockwork

import "time"

type Clock struct{}

func New() *Clock {
	return &Clock{}
}

func (c *Clock) Now() time.Time {
	return time.Now()
}
