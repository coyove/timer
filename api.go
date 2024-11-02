package timer

import (
	"fmt"
	"sync/atomic"
	"time"
	_ "unsafe"
)

var OnError = func(error) {}

var jobsCtr atomic.Int64

func SetInterval1(f func() bool, interval time.Duration) error {
	var j *Job
	var err error
	j, err = SetInterval(func() {
		if !f() {
			ClearInterval(j)
		}
	}, interval)
	return err
}

func SetInterval(f func(), interval time.Duration) (*Job, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("invalid interval: %v", interval)
	}
	j, err := createJob(f, interval, false)
	if err == nil {
		jobsCtr.Add(1)
	}
	return j, err
}

func SetTimeout(f func(), timeout time.Duration) (*Job, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("invalid timeout: %v", timeout)
	}
	j, err := createJob(f, timeout, true)
	if err == nil {
		jobsCtr.Add(1)
	}
	return j, err
}

func (j *Job) Abort() {
	atomic.StoreInt32(&j.dead, 1)
}

func ClearInterval(j *Job) {
	j.Abort()
}

func ClearTimeout(j *Job) {
	j.Abort()
}

func TotalJobs() int {
	return int(jobsCtr.Load())
}
