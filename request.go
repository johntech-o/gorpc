package gorpc

import (
	"sync/atomic"
	"time"
)

const (
	RequestSendOnly   int16 = 1
	MaxPendingRequest int   = 1500
)

type Request struct {
	header       *RequestHeader
	body         interface{}
	writeTimeout time.Duration
	readTimeout  time.Duration
	pending      int32
}

func NewRequest() *Request {
	return &Request{
		pending:      1,
		header:       NewRequestHeader(),
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

func NewRequestHeader() *RequestHeader {
	return &RequestHeader{}
}

func (reqheader *RequestHeader) IsPing() bool {
	if reqheader.Service == "go" && reqheader.Method == "p" {
		return true
	}
	return false
}
