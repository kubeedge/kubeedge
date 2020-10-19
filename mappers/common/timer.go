package mappercommon

import (
	"time"
)

type Timer struct {
	Function func()
	Duration time.Duration
	Times    int
}

func (t *Timer) Start() {
	ticker := time.NewTicker(t.Duration)
	if t.Times > 0 {
		for i := 0; i < t.Times; i++ {
			select {
			case <-ticker.C:
				t.Function()
			}
		}
	} else {
		for {
			select {
			case <-ticker.C:
				t.Function()
			}
		}
	}
}
