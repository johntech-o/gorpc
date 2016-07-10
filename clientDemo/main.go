package main

import (
	"flag"
	"fmt"
	"runtime"
	"sync"
	"time"
	"utility/gorpc"
	"utility/gorpc/data"
	"utility/gorpc/pprof"
)

var client, client2 *gorpc.Client
var I string
var G int
var C int

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	netOptions := gorpc.NewNetOptions(time.Second*10, time.Second*20, time.Second*20)
	client = gorpc.NewClient(netOptions)
	client2 = gorpc.NewClient(netOptions)
}

func main() {
	flag.StringVar(&I, "i", "10.108.87.71:6668", "remote address:port")
	flag.IntVar(&G, "g", 1000000, "amount of goroutine")
	flag.IntVar(&C, "c", 10, "amount of rpc call per goroutine")
	flag.Parse()
	// go func() {

	// 	var lastQps string
	// 	for {
	// 		qps := client.Qps()
	// 		if qps != lastQps {
	// 			fmt.Println("client conn Qps: ", qps)
	// 			lastQps = qps
	// 			var reply string
	// 			if err := client2.CallWithAddress(I, "RpcStatus", "CallStatus", false, &reply); err == nil {
	// 				fmt.Println("-----------------server status: ", reply)
	// 			} else {
	// 				fmt.Println("----------------server call amount error: ", err)
	// 			}
	// 		}
	// 		// pprof.DoMemoryStats(false)
	// 		// pprof.DoProcessStat()
	// 		time.Sleep(time.Second * 5)

	// 	}
	// }()
	EchoStruct(G)
	time.Sleep(3600e9)
}

func EchoStruct(testCount int) {

	var results = struct {
		content map[string]int
		sync.Mutex
	}{content: make(map[string]int)}

	// init time and qps counter
	var counter = CallCalculator{
		fields:         make(map[int]*CallTimer),
		countsPerTimer: C,
	}

	// do not need lock
	clientBook := map[int]chan bool{}
	var wg sync.WaitGroup
	for i := 0; i < testCount; i++ {
		wg.Add(1)
		clientBook[i] = make(chan bool)
		// time.Sleep(1e5)
		go func(count int, start chan bool) {
			defer wg.Done()
			<-start
			counter.Start(count)
			for c := 1; c <= C; c++ {
				var res string
				err := client.CallWithAddress(I, "TestRpcInt", "EchoStruct", data.TestABC{"aaa", "bbb", "ccc"}, &res)
				if err != nil {
					results.Lock()
					results.content[err.Error()] += 1
					results.Unlock()
					return
				}
				results.Lock()
				results.content[res] += 1
				results.Unlock()
			}
			counter.End(count)

		}(i, clientBook[i])
	}
	// start the client to send data
	for i := 0; i < len(clientBook); i++ {
		clientBook[i] <- true
	}
	wg.Wait()

	var reply string
	fmt.Println("\n\n\n-----------------final result-----------------")
	fmt.Println("goroutine num: ", G, " every goroutine call Num: ", C)
	if err := client2.CallWithAddress(I, "RpcStatus", "CallStatus", false, &reply); err == nil {
		fmt.Println("-----------------server status: ", reply)
	} else {
		fmt.Println("----------------server call amount error: ", err)
	}
	fmt.Println("client conn status: ", client.ConnsStatus())
	for result, count := range results.content {
		fmt.Println("------------------TestEchoStruct result and count ", result, count)
		if count < testCount {
			fmt.Println("-------------------have fail call")
		}
	}
	fmt.Println("---------------------time cost: ", counter.timeCost(), "QPS is", counter.QPS())
	fmt.Println("---------------------memstat: ")
	pprof.DoMemoryStats(true)
	fmt.Println("\n\n\n")

}

type CallTimer struct {
	startTime time.Time
	endTime   time.Time
}

type CallCalculator struct {
	sync.Mutex
	fields         map[int]*CallTimer
	countsPerTimer int
}

func (c *CallCalculator) Start(key int) {
	c.Lock()
	c.fields[key] = &CallTimer{startTime: time.Now()}
	c.Unlock()
}

func (c *CallCalculator) End(key int) {
	c.Lock()
	c.fields[key].endTime = time.Now()
	c.Unlock()
}

func (c *CallCalculator) timeCost() time.Duration {
	var minStart time.Time = time.Now().Add(time.Second * 10000)
	var maxEnd time.Time
	c.Lock()
	defer c.Unlock()
	for _, v := range c.fields {
		if v.startTime.Sub(minStart) <= 0 {
			minStart = v.startTime
		}
		if v.endTime.Sub(maxEnd) >= 0 {
			maxEnd = v.endTime
		}
	}
	return maxEnd.Sub(minStart)
}

func (c *CallCalculator) QPS() int {
	timeCost := c.timeCost()
	c.Lock()
	defer c.Unlock()
	return int(float64(len(c.fields)) * float64(c.countsPerTimer) * float64(time.Second) / float64(timeCost))
}
