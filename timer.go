//go:build !linux
// +build !linux

package timer

import (
	"sync/atomic"
	"time"
)

type Job struct {
	f       func()
	t       *time.Timer
	i       time.Duration
	dead    int32
	onetime bool
}

func (j *Job) do() {
	if atomic.LoadInt32(&j.dead) == 1 {
		jobsCtr.Add(-1)
		return
	}
	j.f()
	if j.onetime {
		jobsCtr.Add(-1)
		return
	}
	j.t.Reset(j.i)
}

func createJob(f func(), interval time.Duration, onetime bool) (*Job, error) {
	j := &Job{}
	j.f = f
	j.i = interval
	j.t = time.AfterFunc(interval, j.do)
	j.onetime = onetime
	return j, nil
}
