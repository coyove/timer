package timer

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type Job struct {
	fd       int
	dead     int32
	onetime  bool
	callback func()
}

var eps = make([]struct {
	fd        int
	mu        sync.Mutex
	callbacks map[int]*Job
}, runtime.NumCPU())

var epi atomic.Int64

func init() {
	for i := range eps {
		epfd, err := syscall.EpollCreate1(0)
		if err != nil {
			panic(err)
		}
		eps[i].fd = epfd
		eps[i].callbacks = map[int]*Job{}

		go func(i int) {
			out := make([]syscall.EpollEvent, 100)
			tmp := make([]byte, 8)
			for {
				n, err := syscall.EpollWait(epfd, out, 1000)
				if err != nil {
					if err != syscall.EINTR {
						OnError(err)
						continue
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
					j := eps[i].callbacks[int(ev.Fd)]
					switch {
					case atomic.LoadInt32(&j.dead) == 0:
						j.callback()
						if !j.onetime {
							break
						}
						fallthrough
					default:
						jobsCtr.Add(-1)
						delete(eps[i].callbacks, j.fd)
						if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_DEL, j.fd, nil); err != nil {
							OnError(err)
						}
						if err := syscall.Close(j.fd); err != nil {
							OnError(err)
						}
					}
				}
				eps[i].mu.Unlock()
			}
		}(i)
	}
}

func createJob(f func(), interval time.Duration, onetime bool) (*Job, error) {
	fd, _, err := syscall.RawSyscall(0x11b, 1, 0, 0)
	if err != syscall.Errno(0) {
		return nil, err
	}

	var itimer struct {
		it_interval syscall.Timespec
		it_value    syscall.Timespec
	}

	nano := interval.Nanoseconds()
	sec := nano / 1e9
	itimer.it_value.Sec = sec
	itimer.it_value.Nsec = nano % 1e9
	itimer.it_interval = itimer.it_value

	_, _, err = syscall.RawSyscall6(0x11e, fd, 0, uintptr(unsafe.Pointer(&itimer)), 0, 0, 0)
	if err != syscall.Errno(0) {
		syscall.Close(int(fd))
		return nil, err
	}

	j := &Job{
		callback: f,
		fd:       int(fd),
		onetime:  onetime,
	}
	ep := &eps[epi.Add(1)%int64(len(eps))]
	ep.mu.Lock()
	ep.callbacks[int(fd)] = j
	ep.mu.Unlock()

	var ev syscall.EpollEvent
	ev.Events = syscall.EPOLLIN
	ev.Fd = int32(fd)
	if err := syscall.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, int(fd), &ev); err != nil {
		syscall.Close(int(fd))
		return nil, err
	}
	return j, nil
}
