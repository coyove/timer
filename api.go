package timer

import (
	"fmt"
	"time"
	_ "unsafe"
)

func SetInterval(f func() bool, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("invalid interval: %v", interval)
	}
	return createJob(f, interval)
}
