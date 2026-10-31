package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	return &Limiter{clock: clock, rate: ratePerSec, burst: burst, tokens: float64(burst)}
}

func (l *Limiter) AllowN(n int) bool {
	if n <= 0 || n > l.burst {
		return false
	}
	if l.burst <= 0 || l.clock == nil {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	if l.rate > 0 && l.last.Before(now) {
		l.tokens += min(now.Sub(l.last).Seconds()*l.rate, float64(l.burst))
		l.last = now
	}

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		return true
	}

	return false
}
