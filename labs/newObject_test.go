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

func newObjectFromPool() *Object {
	freeObjects.Lock()
	ob := freeObjects.head
	if ob == nil {
		ob = &Object{}
	} else {
		freeObjects.head = freeObjects.head.next
		*ob = Object{}
	}
	freeObjects.Unlock()
	return ob
}

func freeObject(ob *Object) {
	freeObjects.Lock()
	ob.next = freeObjects.head
	freeObjects.head = ob
	freeObjects.Unlock()
}

// list amount of goroutines and memory occupied
func TestGoroutineCost(t *testing.T) {
	amount := 100000
	pprof.MemStats()
	// fmt.Println(pprof.Current())
	fmt.Printf("create goroutines amount: %d\n", amount)
	for i := 0; i < amount; i++ {
		go func() {
			time.Sleep(time.Second * 10)
		}()
	}
	time.Sleep(time.Second * 2)
	pprof.MemStats()
	pprof.StatIncrement(pprof.TotalAlloc)
	// fmt.Println(pprof.Current())
	pprof.ProcessStats()
	fmt.Print("\n\n")
}

// malloc a object in heap
func expectMallocInHeap(ob *Object) *Object {
	new := &Object{}
	return new
}

// malloc a object in stack
func expectMallocInStack(ob *Object) *Object {
	*ob = Object{}
	// _ = make([]Object, 1000)
	return ob
}

// test malloc in heap or stack function
func TestIfObjectInStack(t *testing.T) {
	ob := &Object{}
	fmt.Println("----------------expect malloc in heap--------------")
	pprof.MemStats()

	for i := 0; i < 10000; i++ {
		s := expectMallocInHeap(ob)
		// never hit this statment use s to prevent compile optimize code
		if s.a == "aaa" {
			fmt.Println("sssss")
		}
	}

	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	fmt.Println("--------------  expect malloc in stack--------------")
	pprof.MemStats()

	for i := 0; i < 10000; i++ {
		expectMallocInStack(ob)
	}
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	pprof.ProcessStats()
	fmt.Print("\n\n")
}

// test malloc use object pool
func TestObjectPool(t *testing.T) {

	pprof.MemStats()
	for i := 0; i < 10000; i++ {
		go func() {
			ob := new(Object)
			ob.a = "ssss"
			time.Sleep(time.Second * 2)
		}()

	}
	time.Sleep(time.Second * 2)
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	for i := 0; i < 10000; i++ {
		go func() {
			ob := newObjectFromPool()
			ob.a = "bbb"
			freeObject(ob)
			time.Sleep(time.Second * 2)
		}()
	}
	time.Sleep(time.Second * 2)
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
}
