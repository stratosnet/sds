package utils

import (
	"sync"
	"time"
)

type LimitRate struct {
	rate       uint64
	interval   time.Duration
	lastAction time.Time
	lock       sync.Mutex
}

func (l *LimitRate) Limit() bool {
	result := false
	for {
		l.lock.Lock()

		if time.Since(l.lastAction) > l.interval {
			l.lastAction = time.Now()
			result = true
		}
		l.lock.Unlock()
		if result {
			return result
		}
		time.Sleep(l.interval)
	}
}

func (l *LimitRate) SetRate(r uint64) {
	l.rate = r
	l.interval = time.Microsecond * time.Duration(1000*1000/(l.rate+45))
	// DebugLog("interval.........", l.interval)
}

func (l *LimitRate) GetRate() uint64 {
	return l.rate
}
