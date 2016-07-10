/*
go test -v github.com/johntech-o/gorpc
*/
package gorpc

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"testing"
	"time"
)

var client *Client

const (
	ExecAmount = 10000
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	go func() {
		log.Println(http.ListenAndServe(":6789", nil))
	}()
	time.Sleep(time.Second * 5)
}

type TestABC struct {
	A, B, C string
}

type TestRpcInt struct {
	i int
}

func (r *TestRpcInt) Update(n int, res *int) error {
	r.i = n
	*res = r.i + 100
	return nil
}

var callTimes = 0

func (r *TestRpcInt) ReturnErr(n int, res *int) error {
	*res = 100
	if n == 1 {
		if callTimes == 0 {
			callTimes++
			return &Error{10000, ErrTypeCanRetry, "user defined retry error"}
		} else {
			return Error{100001, ErrTypeLogic, "after retry user logic error"}
		}
	}
	return errors.New("user defined common error")

}

const EchoContent = "hello echo struct"

func (r *TestRpcInt) EchoStruct(arg TestABC, res *string) error {
	*res = EchoContent
	return nil
}

func TestStartServerClient(t *testing.T) {
	go func() {
		s := NewServer("127.0.0.1:6668")
		s.Register(new(TestRpcInt))
		s.Serve()
		panic("server fail")
	}()
	go func() {
		s := NewServer("127.0.0.1:6669")
		s.Register(new(TestRpcInt))
		s.Serve()
		panic("server fail")
	}()
	// client3 := NewClient(NewNetOptions(time.Second*10, time.Second*20, time.Second*20))
	// var res string
	// err := client3.CallWithAddress("127.0.0.1:6669", "TestRpcInt", "EchoStruct", TestABC{"aaa", "bbb", "ccc"}, &res)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// time.Sleep(1000e9)
	time.Sleep(2e8)
	netOptions := NewNetOptions(time.Second*10, time.Second*20, time.Second*20)
	client = NewClient(netOptions)
	client2 := NewClient(netOptions)
	go func() {
		for {
			var reply string
			if err := client2.CallWithAddress("127.0.0.1:6668", "RpcStatus", "CallStatus", false, &reply); err == nil {
				fmt.Println("server call status: ", reply)
			} else {
				fmt.Println("server call amount error: ", err)
			}
			time.Sleep(time.Second * 5)
			fmt.Println("client conn status: ", client.ConnsStatus())
			fmt.Println("client conn Qps: ", client.Qps())
		}
	}()
}

// 加密之后马上解密
func TestInvalidParams(t *testing.T) {
	var up int
	t.Log("test invalid service")
	e := client.CallWithAddress("127.0.0.1:6668", "xxxx", "Update", 5, &up)
	if e.Errno() == 400 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}

	t.Log("test invalid method")
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "xxxx", 5, &up)
	if e.Errno() == 400 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}

	t.Log("test invalid args")
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "Update", "5", &up)
	if e.Errno() == 400 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}

	t.Log("test invalid reply")
	var upStr string
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "Update", 5, &upStr)
	if e.Errno() == 106 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}

	t.Log("test normal update")
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "Update", 5, &up)
	if e != nil {
		t.Error("fail", e, up)
	} else {
		if up == 105 {
			t.Log("ok", up, e)
		}
	}

	var res int
	t.Log("test remote return can retry error")
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "ReturnErr", 1, &res)
	if e.Errno() == 100001 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}
	t.Log("test remote return error")
	e = client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "ReturnErr", 2, &res)
	if e.Errno() == 500 {
		t.Log("ok", e.Error())
	} else {
		t.Error("fail", e)
	}
}

func TestEchoStruct(t *testing.T) {
	EchoStruct(t, ExecAmount)
}

func EchoStruct(t *testing.T, testCount int) {
	execTimes := testCount
	var results = struct {
		content map[string]int
		sync.Mutex
	}{content: make(map[string]int)}

	var counter = CallCalculator{
		fields: make(map[int]*CallTimer),
	}

	var wg sync.WaitGroup
	for i := 0; i < execTimes; i++ {
		if i > 0 && i%1000000 == 0 {
			time.Sleep(time.Second * 1)
			fmt.Println("___________________________________ current:", i, "amount:", execTimes)
		}
		wg.Add(1)
		// time.Sleep(1e5)
		go func(count int) {
			defer wg.Done()
			var res string
			counter.Start(count)
			err := client.CallWithAddress("127.0.0.1:6668", "TestRpcInt", "EchoStruct", TestABC{"aaa", "bbb", "ccc"}, &res)
			counter.End(count)
			if err != nil {
				results.Lock()
				results.content[err.Error()] += 1
				results.Unlock()
				return
			}
			results.Lock()
			results.content[res] += 1
			results.Unlock()
		}(i)
	}
	wg.Wait()
	for result, count := range results.content {
		fmt.Println("TestEchoStruct result and count ", result, count)
		if count < execTimes {
			t.Error("have fail call")
		}
	}
	fmt.Println("time cost: ", counter.timeCost(), "QPS is", counter.QPS())
	time.Sleep(time.Second * 5)
}

type CallTimer struct {
	startTime time.Time
	endTime   time.Time
}

type CallCalculator struct {
	sync.Mutex
	fields map[int]*CallTimer
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
	var timeCost time.Duration
	c.Lock()
	defer c.Unlock()
	for _, v := range c.fields {
		if v.endTime.Sub(v.startTime) > timeCost {
			timeCost = v.endTime.Sub(v.startTime)
		}
	}
	return timeCost
}

func (c *CallCalculator) QPS() int {
	timeCost := c.timeCost()
	c.Lock()
	defer c.Unlock()
	return int(float64(len(c.fields)) * float64(time.Second) / float64(timeCost))
}
