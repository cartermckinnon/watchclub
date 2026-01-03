package util

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_RateLimitMutex(t *testing.T) {
	d := 100 * time.Millisecond
	n := 10
	m := NewRateLimitedMutex(d)

	start := time.Now()

	wg := sync.WaitGroup{}
	for range n {
		wg.Go(func() {
			m.Lock()
			defer m.Unlock()
		})
	}
	wg.Wait()

	assert.GreaterOrEqual(t, time.Since(start), d*time.Duration(n))
}
