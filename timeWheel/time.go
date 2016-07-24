package timewheel

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	DefaultInterval      = time.Second
	DefaultBucketSize    = 600 // ten min
	DefaultWheelPoolSize = 5
)

// C is receive only chan
type Timer struct {
	C chan struct{}
}

// use default timer wheel init with the default params
func NewTimer(timeout time.Duration) *Timer {
	return DefaultTimerWheel.AddTimer(timeout)
}

// unreference the timer do nothing
func (t *Timer) Stop() {}

// default wheel support max timeout 10min
var DefaultTimerWheel = func() *wheelPool {
	return NewWheelPool(DefaultWheelPoolSize, DefaultInterval, DefaultBucketSize)
}()

type wheelPool struct {
	size   int
	wheels []*wheel
}

// user can New their own Wheel Pool
func NewWheelPool(poolSize int, interval time.Duration, bucketSize int) *wheelPool {
	wp := &wheelPool{
		size: poolSize,
	}
	for i := 0; i < poolSize; i++ {
		wp.wheels = append(wp.wheels, newWheel(i, interval, bucketSize))
	}
	return wp
}

// newTimer to the user
func (wp *wheelPool) NewTimer(timeout time.Duration) *Timer {
	return wp.AddTimer(timeout)
}

// sharding the add lock
func (wp *wheelPool) AddTimer(timeout time.Duration) *Timer {
	index := rand.Intn(wp.size)
	return wp.wheels[index].addTimer(timeout)
}

// close all the wheels
func (wp *wheelPool) Close() {
	for _, w := range wp.wheels {
		w.close()
	}
	return
}

type wheel struct {
	index      int // index in the wheelPool
	interval   time.Duration
	maxTimeout time.Duration
	ticker     *time.Ticker
	closeCh    chan struct{}

	sync.Mutex // protect following
	buckets    []*Timer
	tail       int // current wheel tail index

}

// user define own wheel set the index to zero
func newWheel(index int, interval time.Duration, bucketsSize int) *wheel {
	w := &wheel{
		index:      index,
		interval:   interval,
		maxTimeout: interval * time.Duration(bucketsSize),
		buckets:    make([]*Timer, bucketsSize),
		tail:       0,
		ticker:     time.NewTicker(interval),
		closeCh:    make(chan struct{}, 1),
	}
	for i := range w.buckets {
		w.buckets[i] = &Timer{C: make(chan struct{}, 1)}
	}
	go w.scan()
	return w
}

// add timer to the bucket
func (w *wheel) addTimer(timeout time.Duration) *Timer {
	if timeout <= 0 {
		t := &Timer{C: make(chan struct{}, 1)}
		close(t.C)
		return t
	}
	if timeout >= w.maxTimeout {
		println("exceed maxTimeout set the timeout to MaxTimeout")
		timeout = w.maxTimeout
	}
	fmt.Println(w.index)
	w.Lock()
	index := (w.tail + int(timeout/w.interval)) % len(w.buckets)
	timer := w.buckets[index]
	w.Unlock()
	return timer
}

func (w *wheel) close() {
	close(w.closeCh)
}

// scan the wheel every ticket duration
// close tail timer then notify  which refer to tail timer
// start from the index 1
func (w *wheel) scan() {
	for {
		select {
		case <-w.ticker.C:
			w.Lock()
			w.tail = (w.tail + 1) % len(w.buckets) // move the time pointer ahead
			timer := w.buckets[w.tail]
			w.buckets[w.tail] = &Timer{C: make(chan struct{}, 1)}
			w.Unlock()
			close(timer.C)

		case <-w.closeCh:
			for _, timer := range w.buckets {
				close(timer.C)
			}
			w.ticker.Stop()
			return
		}
	}
}
