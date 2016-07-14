package gorpc

import "sync"

const (
	ReplyTypeData = 0x01
	ReplyTypePong = 0x10
	ReplyTypeAck  = 0x100
)

type ResponseHeader struct {
	Error     *Error
	Seq       uint64
	ReplyType int16
}

type ResponseHeaderSlice struct {
	sync.Mutex
	headers []*ResponseHeader
	connId  string
}

func (rhs *ResponseHeaderSlice) NewResponseHeader() *ResponseHeader {
	rhs.Lock()
	len := len(rhs.headers)
	if len == 0 {
		rhs.Unlock()
		return &ResponseHeader{}
	}
	h := rhs.headers[len-1]
	rhs.headers = rhs.headers[0 : len-1]
	rhs.Unlock()
	return h
}

func (rhs *ResponseHeaderSlice) FreeResponseHeader(rh *ResponseHeader) {
	rh.Error, rh.Seq, rh.ReplyType = nil, 0, 0
	rhs.Lock()
	if len(rhs.headers) == cap(rhs.headers) {
		rhs.Unlock()
		return
	}
	rhs.headers = append(rhs.headers, rh)
	rhs.Unlock()
}

func (respHeader *ResponseHeader) HaveReply() bool {
	return (respHeader.ReplyType & ReplyTypeData) > 0
}

func NewResponseHeader() *ResponseHeader {
	return &ResponseHeader{}
}

type PendingResponse struct {
	connId ConnId
	seq    uint64
	reply  interface{}
	done   chan bool
	err    *Error
}

func NewPendingResponse() *PendingResponse {
	return &PendingResponse{done: make(chan bool, 1)}
}
