package dhash

import (
	"time"

	"github.com/purak/gauss/timenet"
)

type timerServer timenet.Timer

func (self *timerServer) ActualTime(x int, result *time.Time) error {
	*result = (*timenet.Timer)(self).ActualTime()
	return nil
}
