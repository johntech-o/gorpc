package gorpc

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	RequestSendOnly   int16 = 1
	MaxPendingRequest int   = 1000
)

type Request struct {
	header       *RequestHeader
	body         interface{}
	writeTimeout time.Duration
	readTimeout  time.Duration
	pending      int32
}

func NewRequest(header *RequestHeader) *Request {
	return &Request{
		pending:      1,
		header:       header,
		writeTimeout: DefaultWriteTimeout,
	}
}

func (request *Request) IsPending() bool {
	return (atomic.LoadInt32(&request.pending) > 0)
}

func (request *Request) freePending() {
	atomic.AddInt32(&request.pending, -1)
}

type RequestHeader struct {
	Service  string
	Method   string
	Seq      uint64
	CallType int16
}

type RequestHeaderSlice struct {
	sync.Mutex
	headers []*RequestHeader
}

func (rhs *RequestHeaderSlice) NewRequestHeader() *RequestHeader {
	rhs.Lock()
	len := len(rhs.headers)
	if len == 0 {
		rhs.Unlock()
		return &RequestHeader{}
	}
	h := rhs.headers[len-1]
	rhs.headers = rhs.headers[0 : len-1]
	rhs.Unlock()
	return h
}

func (rhs *RequestHeaderSlice) FreeRequestHeader(rh *RequestHeader) {
	rh.Service, rh.Method, rh.Seq, rh.CallType = "", "", 0, 0
	rhs.Lock()
	if len(rhs.headers) == cap(rhs.headers) {
		rhs.Unlock()
		return
	}
	rhs.headers = append(rhs.headers, rh)
	rhs.Unlock()
	return
}
func (reqheader *RequestHeader) IsPing() bool {
	if reqheader.Service == "go" && reqheader.Method == "p" {
		return true
	}
	return false
}
