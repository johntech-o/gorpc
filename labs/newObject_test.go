package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/johntech-o/gorpc/pprof"
)

// new object in goroutines then view gc statistics

type Object struct {
	a    string
	b    int
	c    map[string]string
	d    interface{}
	e    *string
	f    []string
	next *Object
}

type objectPool struct {
	head *Object
	sync.Mutex
}

var freeObjects objectPool

func newObjectInHeap() *Object {
	return &Object{}
}

func newObjectInPool() *Object {
	ob := freeObjects.head
	if ob == nil {
		return &Object{}
	} else {
		freeObjects.head = freeObjects.head.next
		*ob = Object{}
	}
	return ob
}

func resetObject(ob *Object) *Object {

	tmpObs := make([]Object, 1000)
	for i := 0; i <= 1000; i++ {
		tmpObs = append(tmpObs, Object{})
	}
	*ob = tmpObs[0]
	//	fmt.Printf("tmpOb: %p\n", &tmpOb)

	//	fmt.Printf("ob in resetObject func: %p\n", ob)
	return ob
}
func TestGoroutineCost(t *testing.T) {
	//	pprof()
	pprof.DoMemoryStats(true)
	for i := 0; i < 100000; i++ {
		go func() {
			time.Sleep(time.Second * 10)
		}()
	}
	println()
	time.Sleep(time.Second * 10)
	//	pprof()
	pprof.DoMemoryStats(true)
}

func TestIfObjectInStack(t *testing.T) {
	//	pprof()
	//	resetedOb := resetObject(ob)
	//	if resetedOb.a == "" {
	//	}
	//	fmt.Printf("ob: %p,resetedOb:%p,obContest: %v\n", ob, resetedOb, ob)
	for i := 0; i < 1; i++ {
		ob := &Object{a: "aaa", b: 10}
		go resetObject(ob)
		//time.Sleep(time.Microsecond)
	}
	fmt.Println()
	time.Sleep(time.Second)
	//	pprof()
	pprof.DoMemoryStats(true)
}

/*
func pprof() {
	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)
	s := reflect.ValueOf(memoryStats)
	filterFields := map[string]struct{}{
		//		"TotalAlloc": struct{}{},
		"Alloc": struct{}{}, // clear by gc
		//"Mallocs":    struct{}{},
		"Frees": struct{}{},

		"BySize":   struct{}{},
		"Lookups":  struct{}{},
		"Sys":      struct{}{},
		"OtherSys": struct{}{},

		"PauseNs":  struct{}{},
		"PauseEnd": struct{}{},
		// "PauseTotalNs":  struct{}{},
		"LastGC":        struct{}{},
		"NextGC":        struct{}{},
		"NumGC":         struct{}{},
		"GCCPUFraction": struct{}{},
		"GCSys":         struct{}{},
		"EnableGC":      struct{}{},
		"DebugGC":       struct{}{},

		"MCacheSys":   struct{}{},
		"MCacheInuse": struct{}{},
		"BuckHashSys": struct{}{},

		"HeapReleased": struct{}{},
		"HeapSys":      struct{}{},
		"HeapIdle":     struct{}{},
		//"HeapInuse":    struct{}{},
		//"HeapObjects": struct{}{},
		"HeapAlloc": struct{}{},

		//		"StackSys":   struct{}{},
		//		"StackInuse": struct{}{},

		"MSpanSys":   struct{}{},
		"MSpanInuse": struct{}{},
	}
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if _, ok := filterFields[typeOfT.Field(i).Name]; ok {
			continue
		}
		fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
	return
}
*/
