package memPool

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

// 测试给elasticBuf设置属性，获取buf，属性争取
func TestElasticBuf(t *testing.T) {
	pool := New(512, 5)
	buf := NewElasticBuf(3, pool)
	assert("", func() (bool, string) {
		if buf.Index() == 3 {
			return true, "test new elasticBuf success"
		}
		return false, "set index fail"
	})
}

func TestSetGet(t *testing.T) {
	pool := New(512, 5)
	a := pool.Malloc(512)
	a.free()
	b := pool.Malloc(512)
	b.free()
	assert("", func() (bool, string) {
		if a == b {
			return true, "a == b success"
		}
		return false, "not use buffer in pool fail"
	})
	return
}

func TestMallocFree(t *testing.T) {
	pool := New(512, 5)
	a := pool.Malloc(512 - 1)
	a.free()
	a1 := pool.Malloc(512)
	a1.free()
	b := pool.Malloc(512 * 2)
	b.free()
	b1 := pool.Malloc(512*2 + 1)
	b1.free()
	c := pool.Malloc(512*3 + 1)
	c.free()
	d := pool.Malloc(512 * 4)
	d.free()
	e := pool.Malloc(512 * 5)
	e.free()
	f := pool.Malloc(512*5 + 1)
	f.free()
	res := []int64{2, 1, 1, 2, 1, 1}
	assert("", func() (bool, string) {
		for index, count := range pool.Status(true) {
			if res[index] != count {
				return false, "malloc free error not expected result"
			}
		}
		return true, ""
	})
}

// 测试申请一个buf，但使用的时候buf不够用，自动从底层扩展这个buf，再次申请扩展的buf，使用的是之前底层申请的buf
func TestSwapCurrentBuf(t *testing.T) {
	pool := New(512, 5)
	buf1 := pool.Malloc(512)
	readerBuf := make([]byte, 512*2)
	rand.Read(readerBuf)
	reader := bytes.NewBuffer(readerBuf)
	buf1.ReadBytes(reader, 512*2)
	buf1.free()
	buf2 := pool.Malloc(512 * 2)
	buf2.free()
	assert("", func() (bool, string) {
		if buf1 == buf2 {
			return true, "buf1== buf2 success"
		}
		return false, "buf1 != buf2 fail"
	})
}

// monitor the gc trace time
// GODEBUG=gctrace=1 GOGC=20000 ./make 1.4
func TestGCEnableMemBuf(t *testing.T) {
	pool := New(512, 5)
	ebs := map[int]*ElasticBuf{}
	MallocEnablePool(ebs, pool)
	MallocEnablePool(ebs, pool)
	for i := 0; i <= 50000; i++ {
		ebs[i] = pool.Malloc(512)
	}
	ebs = nil
	runtime.GC()
	runtime.GC()
	pool.Status(true)
	// fmt.Println("in use object is", (&runtime.MemProfileRecord).InUseObjects())
}

// func TestGCDisableMemBuf(t *testing.T) {
// 	pool := New(512, 5)
// 	ebs := map[int]*ElasticBuf{}
// 	MallocDisablePool(ebs, pool)
// 	MallocDisablePool(ebs, pool)
// 	pool.Status(true)
// }

func MallocEnablePool(ebs map[int]*ElasticBuf, pool *MemPool) {
	runtime.GC()
	runtime.GC()
	pprof()
	for i := 0; i <= 100000; i++ {
		ebs[i] = pool.Malloc(512)
	}
	for _, b := range ebs {
		b.free()
	}
	runtime.GC()
	runtime.GC()
}

func MallocDisablePool(ebs map[int]*ElasticBuf, pool *MemPool) {
	runtime.GC()
	runtime.GC()
	pprof()
	for i := 0; i <= 100000; i++ {
		ebs[i] = NewElasticBuf(1, pool)
	}
	ebs = make(map[int]*ElasticBuf)
	runtime.GC()
	runtime.GC()
}

func assert(content string, f func() (bool, string)) {
	if content != "" {
		fmt.Println(content)
	}
	isTrue, output := f()
	if isTrue {
		fmt.Println(output)
		return
	}
	fmt.Errorf("error %s", output)
}

func pprof() {
	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)
	s := reflect.ValueOf(memoryStats)
	filterFields := map[string]struct{}{
		// "PauseNs":      struct{}{},
		"PauseEnd":     struct{}{},
		"BySize":       struct{}{},
		"Lookups":      struct{}{},
		"Sys":          struct{}{},
		"LastGC":       struct{}{},
		"NextGC":       struct{}{},
		"OtherSys":     struct{}{},
		"Alloc":        struct{}{},
		"StackSys":     struct{}{},
		"HeapReleased": struct{}{},
		"MSpanInuse":   struct{}{},
		"MCacheSys":    struct{}{},
		"MCacheInuse":  struct{}{},
		"BuckHashSys":  struct{}{},
		"GCSys":        struct{}{},
		"EnableGC":     struct{}{},
		"DebugGC":      struct{}{},
		"HeapSys":      struct{}{},
		"HeapIdle":     struct{}{},
		"HeapInuse":    struct{}{},
		"StackInuse":   struct{}{},
		"MSpanSys":     struct{}{},
		"HeapAlloc":    struct{}{},
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

// type myType struct {
// 	a int
// 	b []byte
// }

// func (mt *myType) Set(n int) {
// 	fmt.Printf("in set pointer address %p src address %p src []byte pointer address %p src []byte address %p\n", &mt, mt, &mt.b, mt.b)
// 	mt.a = n
// 	t := myType{a: 3, b: make([]byte, 10)}
// 	fmt.Printf("in set new object myType address %p []byte pointer address %p []byte address %p\n", &t, &t.b, t.b)
// 	mt.a = t.a
// 	mt.b = t.b
// 	fmt.Printf("in set after swap pointer address %p src address %p src []byte pointer address %p src []byte address %p\n", &mt, mt, &mt.b, mt.b)
// }

// func TestSwap(t *testing.T) {
// 	var p *myType = &myType{a: 0, b: make([]byte, 10)}
// 	fmt.Printf("src pointer address %p src address %p src []byte pointer address %p src []byte address %p\n", &p, p, &p.b, p.b)
// 	p.b = make([]byte, 1000)
// 	fmt.Printf("src pointer address %p src address %p src []byte pointer address %p src []byte address %p\n", &p, p, &p.b, p.b)
// 	p.Set(5)
// 	fmt.Printf("src pointer address %p src address %p src []byte pointer address %p src []byte address %p\n", &p, p, &p.b, p.b)
// }
