package timewheel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

var fn = func(wp *wheelPool, wait time.Duration) bool {
	begin := time.Now()
	t1 := wp.NewTimer(wait)
	select {
	case <-t1.C:
		duration := time.Now().Sub(begin)
		gap := float64(duration-wait) / float64(time.Second)
		if gap < 1 && gap > -1 {
			return true
		}
		fmt.Printf("wait timeout error gap: %0.6f\n", gap)
		return false
	}
}

func TestTimerZero(t *testing.T) {
	if ok := fn(DefaultTimerWheel, 0); ok {
		t.Log("zero success")
	}

}

func TestExceedMaxTimeout(t *testing.T) {
	wp := NewWheelPool(5, time.Second, 6)
	waitTime := time.Second * 8
	if ok := fn(wp, waitTime); !ok {
		t.Log("exceed max success")
	}
}

func TestTimerBatch(t *testing.T) {
	wp := NewWheelPool(5, time.Second, 6)
	var wg sync.WaitGroup
	for i := 1; i <= 6; i++ {
		wg.Add(1)
		go func(wait time.Duration) {
			defer wg.Done()
			if ok := fn(wp, wait); !ok {
				t.Log("wait fail gap too large")
			}
		}(time.Second * time.Duration(i))
	}
	wg.Wait()
}
