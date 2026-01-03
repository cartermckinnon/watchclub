package util

import (
	"sync"
	"time"
)

type RateLimitedMutex interface {
	Lock()
	Unlock()
}

func NewRateLimitedMutex(delay time.Duration) RateLimitedMutex {
	return &rateLimitedMutex{
		m:     sync.Mutex{},
		delay: delay,
	}
}

type rateLimitedMutex struct {
	m     sync.Mutex
	delay time.Duration
}

func (rlm *rateLimitedMutex) Lock() {
	rlm.m.Lock()
}

func (rlm *rateLimitedMutex) Unlock() {
	time.Sleep(rlm.delay)
	rlm.m.Unlock()
}
