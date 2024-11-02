package timer

import (
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

func TestBench(t *testing.T) {
	const N = 1e4
	var ss, cc, ii atomic.Int64
	for i := 0; i < N; i++ {
		time.AfterFunc(time.Duration(rand.Intn(100))*time.Millisecond, func() {
			var sum, c int64
			start := time.Now().UnixNano()
			SetInterval(func() bool {
				now := time.Now().UnixNano()
				sum += now - start
				c++
				start = now
				if c < 400 {
					return true
				}
				ss.Add(sum)
				cc.Add(c)
				ii.Add(1)
				return false
			}, 20e6)
		})
	}
	for ii.Load() != N {
		time.Sleep(time.Second)
		fmt.Println(ii.Load(), "total jobs:", TotalJobs())
	}

	ai := ss.Load() / cc.Load()
	if diff := math.Abs(float64(ai-20e6) / 20e6); diff > 0.01 {
		t.Fatal(ai)
	}
	t.Log("interval=", ai)
}
