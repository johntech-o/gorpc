package calculator

import (
	"time"
	"sync"
	"sort"
	"fmt"
)

type CallTimer struct {
	id        uint64
	startTime time.Time
	endTime   time.Time
}

type CallCalculator struct {
	sync.Mutex
	id           int
	Fields       map[int]*CallTimer
	FieldsSorted []*CallTimer
	RangeResult  map[float64]time.Duration // ratio:qps
}

var SummaryRatio = []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}

func NewCallCalculator() *CallCalculator {
	c := &CallCalculator{
		Fields:       make(map[int]*CallTimer, 1000000),
		FieldsSorted: make([]*CallTimer, 0, 1000000),
		RangeResult:  map[float64]time.Duration{},
	}
	for _, ratio := range SummaryRatio {
		c.RangeResult[ratio] = 0
	}
	return c
}

func (c CallCalculator) Len() int { return len(c.FieldsSorted) }

func (c CallCalculator) Swap(i, j int) {
	c.FieldsSorted[i], c.FieldsSorted[j] = c.FieldsSorted[j], c.FieldsSorted[i]
}
func (c CallCalculator) Less(i, j int) bool {
	return c.FieldsSorted[i].endTime.Sub(c.FieldsSorted[i].startTime) < c.FieldsSorted[j].endTime.Sub(c.FieldsSorted[j].startTime)
}

func (c *CallCalculator) Start() (index int) {
	c.Lock()
	index = c.id
	c.Fields[c.id] = &CallTimer{startTime: time.Now()}
	c.id++
	c.Unlock()
	return
}

func (c *CallCalculator) End(index int) {
	c.Lock()
	c.Fields[index].endTime = time.Now()
	c.Unlock()
}

func (c *CallCalculator) sort() {
	if len(c.FieldsSorted) == 0 {
		for _, v := range c.Fields {
			c.FieldsSorted = append(c.FieldsSorted, v)
		}
		sort.Sort(c)
	}
}

func (c *CallCalculator) Summary() {
	c.sort()

	var timeCost time.Duration
	indexToCal := make(map[int]float64)
	for ratio, _ := range c.RangeResult {
		index := int(float64(len(c.FieldsSorted)) * ratio)
		indexToCal[index-1] = ratio
	}

	minStartTime, maxEndTime := time.Now(), time.Time{}

	for index, v := range c.FieldsSorted {
		if v.startTime.Before(minStartTime) {
			minStartTime = v.startTime
		}
		if v.endTime.After(maxEndTime) {
			maxEndTime = v.endTime
		}
		if v.endTime.Sub(v.startTime) > timeCost {
			timeCost = v.endTime.Sub(v.startTime)
		}
		if ratio, ok := indexToCal[index]; ok {
			c.RangeResult[ratio] = timeCost
		}
	}

	for _, ratio := range SummaryRatio {
		timeCost = c.RangeResult[ratio]
		callsRatio := int(100 * ratio)
		maxTimeCost := int(timeCost / time.Millisecond)

		fmt.Printf("%3d%% calls consume less than %d ms \n", callsRatio, maxTimeCost)
	}
	costSeconds := int64(maxEndTime.Sub(minStartTime)) / int64(time.Second)
	qps := int64(len(c.FieldsSorted)) * int64(time.Second) / int64(maxEndTime.Sub(minStartTime))
	fmt.Printf("request amount: %d, cost times : %d second, average Qps: %d \n", len(c.FieldsSorted), costSeconds, qps)
}
