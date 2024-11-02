package timer

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type job struct {
	c        *time.Ticker
	callback func() bool
	t        time.Time
}

var eps = make([]struct {
	fd        int
	mu        sync.Mutex
	callbacks map[int]*job
}, runtime.NumCPU())

var epi atomic.Int64

func init() {
	for i := range eps {
		epfd, err := syscall.EpollCreate1(0)
		if err != nil {
			panic(err)
		}
		eps[i].fd = epfd
		eps[i].callbacks = map[int]*job{}

		go func(i int) {
			out := make([]syscall.EpollEvent, 100)
			tmp := make([]byte, 8)
			for {
				n, err := syscall.EpollWait(epfd, out, 1000)
				if err != nil {
					if err != syscall.EINTR {
						panic(err)
					}
					if n <= 0 {
						continue
					}
				}
				res := out[:n]
				for _, ev := range res {
					syscall.Read(int(ev.Fd), tmp)
				}

				eps[i].mu.Lock()
				for _, ev := range res {
					ok := eps[i].callbacks[int(ev.Fd)].callback()
					if !ok {
						delete(eps[i].callbacks, int(ev.Fd))
						if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_DEL, int(ev.Fd), nil); err != nil {
							panic(err)
						}
						if err := syscall.Close(int(ev.Fd)); err != nil {
							panic(err)
						}
					}
				}
				eps[i].mu.Unlock()
			}
		}(i)
	}
}

func createJob(f func() bool, interval time.Duration) error {
	fd, _, err := syscall.RawSyscall(0x11b, 1, 0, 0)
	if err != syscall.Errno(0) {
		return err
	}

	var itimer struct {
		it_interval syscall.Timespec
		it_value    syscall.Timespec
	}

	itimer.it_value.Nsec = interval.Nanoseconds()
	itimer.it_interval.Nsec = interval.Nanoseconds()

	_, _, err = syscall.RawSyscall6(0x11e, fd, 0, uintptr(unsafe.Pointer(&itimer)), 0, 0, 0)
	if err != syscall.Errno(0) {
		syscall.Close(int(fd))
		return err
	}

	ep := &eps[epi.Add(1)%int64(len(eps))]
	ep.mu.Lock()
	ep.callbacks[int(fd)] = &job{
		callback: f,
	}
	ep.mu.Unlock()

	var ev syscall.EpollEvent
	ev.Events = syscall.EPOLLIN
	ev.Fd = int32(fd)
	if err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, int(fd), &ev); err != nil {
		syscall.Close(int(fd))
		return err
	}
	return nil
}

func TotalJobs() (s int) {
	for i := range eps {
		eps[i].mu.Lock()
		s += len(eps[i].callbacks)
		eps[i].mu.Unlock()
	}
	return
}
