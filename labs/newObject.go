package main

import (
	"fmt"
	"sync"
	"testing"
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
	tmpOb := Object{}
	fmt.Printf("tmpOb: %p", &tmpOb)
	*ob = tmpOb
	fmt.Printf("ob in resetObject func: %p", ob)
	return ob
}

func TestIfObjectInStack(t testing.T) {
	ob := &Object{a: "aaa", b: 10}
	resetedOb := resetObject(ob)
	fmt.Printf("ob: %p,resetedOb:%p", ob, resetedOb)
}
