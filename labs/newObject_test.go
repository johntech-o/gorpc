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
	fmt.Println(pprof.Current())
	fmt.Printf("create goroutines amount: %d\n", amount)
	for i := 0; i < amount; i++ {
		go func() {
			time.Sleep(time.Second * 10)
		}()
	}
	time.Sleep(time.Second * 2)
	pprof.MemStats()
	pprof.StatIncrement(pprof.TotalAlloc)
	fmt.Println(pprof.Current())
	pprof.ProcessStats()
}

// malloc a object in heap
func expectMallocInHeap(ob *Object) *Object {
	new := &Object{}
	return new
}

// malloc a object in stack
func expectMallocInStack(ob *Object) *Object {
	*ob = Object{}
	return ob
}

func TestIfObjectInStack(t *testing.T) {
	ob := &Object{}
	fmt.Println("----------------expect malloc in heap--------------")
	pprof.MemStats()
	for i := 0; i < 100000; i++ {
		s := expectMallocInHeap(ob)
		if s.a == "aaa" {
			fmt.Println("sssss")
		}
	}
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects)
	fmt.Println("------------------------------expect malloc in stack")
	pprof.MemStats()
	for i := 0; i < 100000; i++ {
		expectMallocInStack(ob)
	}
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects)
	pprof.ProcessStats()
}
