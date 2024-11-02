//go:build !linux
// +build !linux

package timer

import (
	"sync/atomic"
	"time"
)

type Job struct {
	f    func()
	t    *time.Timer
	i    time.Duration
	dead int64
}

func (j *Job) do() {
	if atomic.LoadInt64(&j.dead) == 1 {
		return
	}
	j.f()
	j.t.Reset(j.i)
}

func createJob(f func(), interval time.Duration) (*Job, error) {
	j := &Job{}
	j.f = f
	j.i = interval
	j.t = time.AfterFunc(interval, j.do)
	return j, nil
}
