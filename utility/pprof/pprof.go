package pprof

import (
	"fmt"
	"reflect"
	"runtime"
)

func init() {
	Proc.Init()
}

const (
	TotalAlloc   = "TotalAlloc"
	HeapObjects  = "HeapObjects"
	StackInuse   = "StackInuse"
	NumGC        = "NumGC"
	PauseTotalMs = "PauseTotalMs"
)

type Stats struct {
	index                int
	content              map[int]map[string]int
	lastNumGc            uint32
	maxGoroutineNum      int
	lastLoopGoroutineNum int
}

var memStats = func() *Stats {
	return &Stats{content: make(map[int]map[string]int)}
}()

func MemStats() {
	memStats.DoMemoryStats()
}

func ProcessStats() {
	memStats.DoProcessStat()
}

// if gc happend
func (s *Stats) DoMemoryStats() {

	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)
	v := reflect.ValueOf(memoryStats)
	t := reflect.TypeOf(memoryStats)
	if s.lastNumGc != memoryStats.NumGC {
		s.lastNumGc = memoryStats.NumGC
	}
	s.content[s.index] = make(map[string]int)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		switch fieldType.Name {
		case "TotalAlloc", "HeapAlloc", "HeapObjects", "Mallocs", "StackInuse", "MSpanInuse", "NumGC":
			v := field.Interface()
			switch t := v.(type) {
			case uint32:
				s.content[s.index][fieldType.Name] = int(v.(uint32))
			case uint64:
				s.content[s.index][fieldType.Name] = int(v.(uint64))
			default:
				fmt.Println(t)
			}
		case "PauseNs":
			s.content[s.index]["PauseMs"] = int(memoryStats.PauseNs[(memoryStats.NumGC+255)%256]) / 1e6
		case "PauseTotalNs":
			s.content[s.index]["PauseTotalMs"] = int(field.Interface().(uint64)) / 1e6
		}
	}
	s.index += 1
}

func StatIncrement(keys ...string) {
	for _, key := range keys {
		if memStats.index == 1 {
			fmt.Printf("%s: %d - %d = %d\n", key, memStats.content[memStats.index-1][key], 0, memStats.content[memStats.index-1][key])
			continue
		}
		// fmt.Println(memStats.index-1, memStats.index-2)
		// fmt.Println(memStats.content[memStats.index-1])
		// fmt.Println(memStats.content[memStats.index-2])
		res := memStats.content[memStats.index-1][key] - memStats.content[memStats.index-2][key]
		fmt.Printf("%s: %d - %d = %d\n", key, memStats.content[memStats.index-1][key], memStats.content[memStats.index-2][key], res)
	}

}

func Current() string {
	result := "---------------NumGC memStats.lastNumGc ------------------\n"
	result = fmt.Sprintf("%v\n", memStats.content)
	result += "-------------------------------------------------------"
	return result
}

func (s *Stats) DoProcessStat() {
	n := runtime.NumGoroutine()
	if n > s.maxGoroutineNum {
		s.maxGoroutineNum = n
	}
	if n != s.lastLoopGoroutineNum {
		fmt.Printf("current goutines:%10d, max goroutine num: %10d\n", n, s.maxGoroutineNum)
		s.lastLoopGoroutineNum = n
		Proc.Cpu.Refresh()
		Proc.Mem.Refresh()
		fmt.Println("process cpu and mem: ", Proc.Cpu, Proc.Mem)
	}
}
