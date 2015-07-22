package gorpc

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

type ClientStatus struct {
	idleAmount     uint64
	workingAmount  uint64
	creatingAmount uint64
	readAmount     uint64
}

func (cs *ClientStatus) IncreReadAmount() {
	atomic.AddUint64(&cs.readAmount, 1)
}

func (cs *ClientStatus) ReadAmount() uint64 {
	return atomic.LoadUint64(&cs.readAmount)
}

type ServerStatus struct {
	CallAmount  uint64
	WriteAmount uint64
	ErrorAmount uint64
	ReadBytes   uint64
	WriteBytes  uint64
}

type ServerStatusPerSecond struct {
	Result map[string]uint64
	Errno  int
}

func (ss *ServerStatus) IncrCallAmount() {
	atomic.AddUint64(&ss.CallAmount, 1)
}

func (ss *ServerStatus) IncrErrorAmount() {
	atomic.AddUint64(&ss.ErrorAmount, 1)
}

func (ss *ServerStatus) IncrReadBytes(bytes uint64) {
	atomic.AddUint64(&ss.ReadBytes, bytes)
}

func (ss *ServerStatus) IncrWriteBytes(bytes uint64) {
	atomic.AddUint64(&ss.WriteBytes, bytes)
}

// detail calculate tx,rx,qps/s
func (ss *ServerStatus) String() string {
	status := ss.Status()
	result, err := json.Marshal(status)
	if err != nil {
		return `{"Result":"inner error marshal error ` + err.Error() + `","Errno":500}`
	}
	return string(result)
}

// detail calculate tx,rx,qps/s
func (ss *ServerStatus) Status() *ServerStatusPerSecond {
	var status = ServerStatusPerSecond{Result: make(map[string]uint64)}

	callAmount := atomic.LoadUint64(&ss.CallAmount)
	errAmount := atomic.LoadUint64(&ss.ErrorAmount)
	readBytes := atomic.LoadUint64(&ss.ReadBytes)
	writeBytes := atomic.LoadUint64(&ss.WriteBytes)
	time.Sleep(time.Second)
	status.Result["CallAmount"] = atomic.LoadUint64(&ss.CallAmount)
	status.Result["ErrorAmount"] = atomic.LoadUint64(&ss.ErrorAmount)
	status.Result["ReadBytes"] = atomic.LoadUint64(&ss.ReadBytes)
	status.Result["WriteBytes"] = atomic.LoadUint64(&ss.WriteBytes)
	status.Result["Call/s"] = status.Result["CallAmount"] - callAmount
	status.Result["Err/s"] = status.Result["ErrorAmount"] - errAmount
	status.Result["ReadBytes/s"] = status.Result["ReadBytes"] - readBytes
	status.Result["WriteBytes/s"] = status.Result["WriteBytes"] - writeBytes
	return &status
}
