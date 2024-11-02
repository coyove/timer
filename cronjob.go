//go:build !linux
// +build !linux

package timer

import (
	"sync/atomic"
	"time"
)

var jobsCtr atomic.Int64

func createJob(f func() bool, interval time.Duration) error {
	go func() {
		jobsCtr.Add(1)
		defer jobsCtr.Add(-1)

		t := time.NewTicker(interval)
		defer t.Stop()

		for range t.C {
			if !f() {
				break
			}
		}
	}()
	return nil
}

func TotalJobs() int {
	return int(jobsCtr.Load())
}
