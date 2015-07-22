package pprof

import (
	"fmt"
	"reflect"
	"runtime"
)

func init() {
	Proc.Init()
}

var lastNumGc uint32

// if gc happend
func DoMemoryStats(output bool) {
	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)
	v := reflect.ValueOf(memoryStats)
	t := reflect.TypeOf(memoryStats)
	if lastNumGc != memoryStats.NumGC {
		output = true
		lastNumGc = memoryStats.NumGC
	}
	if output {
		fmt.Println("---------------NumGC", memoryStats.NumGC, "------------------")
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)
			switch fieldType.Name {
			case "TotalAlloc", "HeapAlloc", "HeapObjects", "Mallocs", "StackInuse", "MSpanInuse", "NumGC":
				fmt.Println(fieldType.Name, field.Interface())
			case "PauseNs":
				fmt.Println("PauseMs", float64(memoryStats.PauseNs[(memoryStats.NumGC+255)%256])/1e6)
			case "PauseTotalNs":
				fmt.Println("PauseTotalMs", float64(field.Interface().(uint64))/1e6)
			}
		}
		fmt.Println("---------------------------------")
	}
}

var maxGoroutineNum int
var lastLoopGoroutineNum int

func DoProcessStat() {
	n := runtime.NumGoroutine()
	if n > maxGoroutineNum {
		maxGoroutineNum = n
	}
	if n != lastLoopGoroutineNum {
		fmt.Println("current goutines:", n)
		fmt.Println("max goroutine num: ", maxGoroutineNum)
		lastLoopGoroutineNum = n
		Proc.Cpu.Refresh()
		Proc.Mem.Refresh()
		fmt.Println("process cpu and mem: ", Proc.Cpu, Proc.Mem)
	}
}
