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

// use by client
var clientConnId ConnId

// user by server
var serverConnId ConnId

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

	timeLock       sync.RWMutex // protects followinng
	readDeadline   time.Time
	writeDeadline  time.Time
	closeByTimerGC bool
}

// for flow control
func (conn *Connection) Write(p []byte) (n int, err error) {
	n, err = conn.TCPConn.Write(p)
	conn.server.status.IncrWriteBytes(uint64(n))
	return
}

// for flow control
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
	return &ConnDriver{
		TCPConn:          conn,
		ConnId:           serverConnId.Incr(),
		writeBuf:         buf,
		dec:              gob.NewDecoder(c),
		enc:              gob.NewEncoder(buf),
		exitWriteNotify:  make(chan bool, 1),
		pendingResponses: make(map[uint64]*PendingResponse),
		pendingRequests:  make(chan *Request, MaxPendingRequest),
		// timer-gc to close timeout socket
		readDeadline:  time.Now() + DefaultReadTimeout,
		writeDeadline: time.Now() + DefaultWriteTimeout,
	}
}

func (conn *ConnDriver) Sequence() uint64 {
	conn.sequence += 1
	return conn.sequence
}

// timer-gc to close timeout socket
// deadlock-review conn.timeLock.Lock
func (conn *ConnDriver) SetReadDeadline(time time.Time) (err error) {
	conn.timeLock.Lock()
	if !conn.isCloseByGCTimer {
		if time > conn.readDeadline {
			conn.readDeadline = time
		}
	} else {
		err = ErrNetTimerGCArrive
	}
	conn.timeLock.Unlock()
	return
}

// timer-gc to close timeout socket
// deadlock-review conn.timeLock.Lock
func (conn *ConnDriver) SetWriteDeadline(time time.Time) (err error) {
	conn.timeLock.Lock()
	if !conn.isCloseByGCTimer {
		if time > conn.writeDeadline {
			conn.writeDeadline = time
		}
	} else {
		err = ErrNetTimerGCArrive
	}
	conn.timeLock.Unlock()
	return
}

// deadlock-review conn.timeLock.RLock
func (conn *ConnDriver) isCloseByGCTimer() bool {
	conn.timeLock.RLock()
	isLocked := conn.isCloseByGCTimer
	conn.timeLock.RUnlock()
	return isLocked
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
