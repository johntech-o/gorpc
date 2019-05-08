package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/johntech-o/gorpc/utility/pprof"
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
	head  *Object
	count int
	sync.Mutex
}

var freeObjects objectPool

func newObjectInHeap() *Object {
	return &Object{}
}

func lenPool() int {
	freeObjects.Lock()
	count := freeObjects.count
	freeObjects.Unlock()
	return count
}

func newObjectFromPool() *Object {
	freeObjects.Lock()
	ob := freeObjects.head
	if ob == nil {
		ob = &Object{}
	} else {
		freeObjects.head = freeObjects.head.next
		freeObjects.count -= 1
		//		*ob = Object{}
	}
	freeObjects.Unlock()
	return ob
}

func freeObject(ob *Object) {
	freeObjects.Lock()
	ob.next = freeObjects.head
	freeObjects.head = ob
	freeObjects.count += 1
	freeObjects.Unlock()
}

// list amount of goroutines and memory occupied
func TestGoroutineCost(t *testing.T) {
	amount := 100000
	pprof.MemStats()
	fmt.Println(pprof.Current())
	fmt.Printf("create goroutines amount: %d\n", amount)
	var wg sync.WaitGroup
	for i := 0; i < amount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Second * 10)
		}()
	}
	wg.Wait()
	fmt.Println(pprof.Current())
	pprof.ProcessStats()

	pprof.MemStats()
	pprof.StatIncrement(pprof.TotalAlloc)
	fmt.Print("\n\n")
}

// test malloc in heap or stack function
func TestIfObjectInStack(t *testing.T) {
	amount := 10000
	for i := 0; i < amount; i++ {
		ob := Object{}
		if ob.a == "aaa" {
			ob.a = "bbb"
		}
		freeObject(&ob)
	}
	fmt.Println(lenPool())

	ob := &Object{}
	fmt.Println("----------------expect malloc in heap--------------")
	pprof.MemStats()

	for i := 0; i < amount; i++ {
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

	for i := 0; i < amount; i++ {
		expectMallocInStack(ob)
	}
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	pprof.ProcessStats()

	fmt.Println("----------------expect malloc in mem pool--------------")
	pprof.MemStats()
	for i := 0; i < amount; i++ {
		s := newObjectFromPool()
		// never hit this statment use s to prevent compile optimize code
		if s.a == "aaa" {
			fmt.Println("sssss")
		}
		freeObject(s)
	}
	pprof.MemStats()
	pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	fmt.Print("\n\n")
}

// test malloc use object pool
func TestObjectPool(t *testing.T) {
	time.Sleep(time.Second)
	amount := 1000
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		fmt.Println("--------------malloc object without object pool-------------")
		pprof.MemStats()
		for i := 0; i < amount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 1000; i++ {
					ob := new(Object)
					if ob.a == "aaa" {
						ob.a = "bbb"
					}
				}

			}()

		}
		wg.Wait()
		pprof.MemStats()
		pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	}
	for i := 0; i < 5; i++ {
		fmt.Println("----------------malloc object use object pool------------ ")
		pprof.MemStats()
		for i := 0; i < amount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 1000; i++ {
					ob := newObjectFromPool()
					if ob.a == "aaa" {
						ob.a = "bbb"
					}
					freeObject(ob)
				}

			}()
		}
		wg.Wait()
		pprof.MemStats()
		pprof.StatIncrement(pprof.HeapObjects, pprof.TotalAlloc)
	}
	fmt.Println("\n\n")
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
