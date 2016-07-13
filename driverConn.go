package gorpc

import (
	"bufio"
	"container/list"
	"encoding/gob"
	// "fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ConnId uint64

func (this *ConnId) Incr() ConnId {
	return ConnId(atomic.AddUint64((*uint64)(this), 1))
}

var clientConnId ConnId

type Connection struct {
	*net.TCPConn
	server *Server
}

func NewConnection(conn *net.TCPConn, server *Server) *Connection {
	return &Connection{conn, server}
}

type ConnDriver struct {
	*net.TCPConn
	writeBuf         *bufio.Writer
	dec              *gob.Decoder
	enc              *gob.Encoder
	exitWriteNotify  chan bool
	pendingRequests  chan *Request
	sync.Mutex       // protects following
	pendingResponses map[uint64]*PendingResponse
	connId           ConnId
	netError         error
	sequence         uint64
	lastUseTime      time.Time
	callCount        int // for priority
	workingElement   *list.Element
	idleElement      *list.Element
	*RequestHeaderSlice
	*ResponseHeaderSlice
}

func (conn *Connection) Write(p []byte) (n int, err error) {
	n, err = conn.TCPConn.Write(p)
	conn.server.status.IncrWriteBytes(uint64(n))
	return
}

func (conn *Connection) Read(p []byte) (n int, err error) {
	n, err = conn.TCPConn.Read(p)
	conn.server.status.IncrReadBytes(uint64(n))
	return
}

func NewConnDriver(conn *net.TCPConn, server *Server) *ConnDriver {
	var c io.ReadWriter
	if server != nil {
		c = NewConnection(conn, server)
	} else {
		c = conn
	}
	buf := bufio.NewWriter(c)
	cd := &ConnDriver{
		TCPConn:          conn,
		writeBuf:         buf,
		dec:              gob.NewDecoder(c),
		enc:              gob.NewEncoder(buf),
		exitWriteNotify:  make(chan bool, 1),
		pendingResponses: make(map[uint64]*PendingResponse),
		pendingRequests:  make(chan *Request, MaxPendingRequest),
	}
	// server end only use responseHeaderSlice as objectPool whenerver  a client end just use requestHeaderSlice
	if server != nil {
		cd.ResponseHeaderSlice = &ResponseHeaderSlice{headers: make([]*ResponseHeader, 500)[0:0]}
	} else {
		cd.RequestHeaderSlice = &RequestHeaderSlice{headers: make([]*RequestHeader, 500)[0:0]}
	}
	return cd

}

func (conn *ConnDriver) Sequence() uint64 {
	conn.sequence += 1
	return conn.sequence
}

func (conn *ConnDriver) ReadRequestHeader(reqHeader *RequestHeader) error {
	return conn.dec.Decode(reqHeader)
}

func (conn *ConnDriver) ReadRequestBody(body interface{}) error {
	return conn.dec.Decode(body)
}

func (conn *ConnDriver) WriteResponseHeader(respHeader *ResponseHeader) error {
	return conn.enc.Encode(respHeader)
}

func (conn *ConnDriver) WriteResponseBody(body interface{}) error {
	return conn.enc.Encode(body)
}

func (conn *ConnDriver) WriteRequestHeader(ReqHeader *RequestHeader) error {
	return conn.enc.Encode(ReqHeader)
}

func (conn *ConnDriver) WriteRequestBody(body interface{}) error {
	return conn.enc.Encode(body)
}

func (conn *ConnDriver) FlushWriteToNet() error {
	return conn.writeBuf.Flush()
}

func (conn *ConnDriver) ReadResponseHeader(RespHeader *ResponseHeader) error {
	return conn.dec.Decode(RespHeader)
}

func (conn *ConnDriver) ReadResponseBody(body interface{}) error {
	return conn.dec.Decode(body)
}

func (conn *ConnDriver) PendingResponseCount() int {
	return len(conn.pendingResponses)
}
func (conn *ConnDriver) AddPendingResponse(pr *PendingResponse) {
	conn.pendingResponses[pr.seq] = pr
}

func (conn *ConnDriver) RemovePendingResponse(seq uint64) *PendingResponse {
	if conn.pendingRequests == nil {
		return nil
	}
	if presp, ok := conn.pendingResponses[seq]; ok {
		delete(conn.pendingResponses, seq)
		return presp
	}
	return nil
}

func (conn *ConnDriver) ClearPendingResponses() map[uint64]*PendingResponse {
	presps := conn.pendingResponses
	conn.pendingResponses = nil
	return presps
}

func (conn *ConnDriver) AddPendingRequest(request *Request) error {
	select {
	case conn.pendingRequests <- request:
		return nil
	default:
		return ErrPendingRequestFull
	}
}
