package gorpc

import (
	"container/list"
	"net"
	"sync"
	"time"
)

const (
	TimeWheelBucketSize = 600
	TimeWheelInterval   = time.Second
)

type ConnPool struct {
	address       string
	sync.Mutex    // protects following fields
	openConnsPool *OpensPool
	maxOpenConns  int
	maxIdleConns  int
	creatingConns int
	client        *Client
	status        *ClientStatus
}

// new connection pool and start async-ping goroutine and timer-garbage-collect goroutine
func NewConnPool(address string, maxOpenConns, maxIdleConns int) *ConnPool {
	cp := &ConnPool{
		openConnsPool: NewOpenPool(),
		maxOpenConns:  maxOpenConns,
		maxIdleConns:  maxIdleConns,
		address:       address,
		status:        &ClientStatus{},
	}
	go cp.ServeIdlePing()
	go cp.GCTimer()
	return cp
}

func (cp *ConnPool) poolStatus() *ClientStatus {
	cp.Lock()
	workingAmount := cp.openConnsPool.workingList.Len()
	idleAmount := cp.openConnsPool.idleList.Len()
	creatingAmount := cp.creatingConns
	cp.Unlock()
	return &ClientStatus{
		uint64(idleAmount),
		uint64(workingAmount - idleAmount),
		uint64(creatingAmount),
		cp.status.ReadAmount(),
	}
}

func (cp *ConnPool) connect(address string, connectTimeout time.Duration) (*net.TCPConn, error) {
	c, err := net.DialTimeout("tcp", address, connectTimeout)
	if err != nil {
		return nil, err
	}
	return c.(*net.TCPConn), nil
}

func (cp *ConnPool) createConn(connectTimeout time.Duration) (*ConnDriver, *Error) {
	conn, err := cp.connect(cp.address, connectTimeout)
	if err == nil {
		var rpcConn *ConnDriver = NewConnDriver(conn, nil)
		rpcConn.connId = clientConnId.Incr()
		go cp.serveRead(rpcConn)
		go cp.serveWrite(rpcConn)
		cp.Lock()
		cp.creatingConns--
		cp.openConnsPool.WorkingPushBack(rpcConn)
		cp.Unlock()
		return rpcConn, nil
	}
	cp.Lock()
	cp.creatingConns--
	cp.Unlock()
	return nil, ErrNetConnectFail.SetReason(err.Error())
}

func (cp *ConnPool) Conn(connectTimeout time.Duration, useOpenedConn bool) (*ConnDriver, *Error) {
	var rpcConn *ConnDriver
	var err *Error
	cp.Lock()
	// cannot use defer createConn cause block
	if rpcConn, err = cp.IdleConn(); err == nil {
		cp.Unlock()
		return rpcConn, nil
	}
	if useOpenedConn {
		if rpcConn, err = cp.WorkingConn(); err == nil {
			cp.Unlock()
			return rpcConn, nil
		}
		cp.Unlock()
		return nil, ErrNoWorkingConn.SetReason("use opened conn fail" + err.Error())
	}
	if (cp.openConnsPool.Len() + cp.creatingConns) < cp.maxOpenConns {
		cp.creatingConns++
		cp.Unlock()
		rpcConn, err := cp.createConn(connectTimeout) // add to working slice and creatingConns--
		return rpcConn, err
	}
	if cp.openConnsPool.Len() == cp.maxOpenConns {
		if rpcConn, err = cp.WorkingConn(); err == nil {
			cp.Unlock()
			return rpcConn, nil
		}
	}
	cp.Unlock()
	// the pool is createing connection ,wait
	timer := time.NewTimer(connectTimeout)
	defer timer.Stop()
	for {
		cp.Lock()
		if rpcConn, err = cp.WorkingConn(); err == nil {
			cp.Unlock()
			return rpcConn, nil
		}
		select {
		case <-timer.C:
			cp.Unlock()
			return nil, ErrCallConnectTimeout
		default:
		}
		cp.Unlock()
		time.Sleep(2e8)
	}
}

func (cp *ConnPool) IdleConn() (*ConnDriver, *Error) {
	conn := cp.openConnsPool.IdlePopFront()
	if conn == nil {
		return nil, ErrNoIdleConn
	}
	return conn, nil
}

func (cp *ConnPool) WorkingConn() (*ConnDriver, *Error) {
	rpcConn := cp.openConnsPool.WorkingMoveFrontToBack()
	if rpcConn == nil {
		return nil, ErrNoWorkingConn
	}
	return rpcConn, nil
}

func (cp *ConnPool) MarkAsIdle(conn *ConnDriver) {
	cp.openConnsPool.IdlePushBack(conn)
}

func (cp *ConnPool) RemoveConn(conn *ConnDriver) {
	cp.openConnsPool.RemoveFromList(conn)
}

// serve read wait response from server
func (cp *ConnPool) serveRead(rpcConn *ConnDriver) {
	var err error
	for {
		// if write encode error, set err info from write goroutine
		rpcConn.Lock()
		if rpcConn.netError != nil {
			err = rpcConn.netError
			rpcConn.Unlock()
			break
		}
		rpcConn.Unlock()
		respHeader := NewResponseHeader()
		// pipeline so use the const read timeout
		if err = rpcConn.SetReadDeadline(time.Now().Add(DefaultClientWaitResponseTimeout)); err != nil {
			break
		}
		if err = rpcConn.ReadResponseHeader(respHeader); err != nil {
			// println("read response header error: ", err.Error())
			break
		}
		cp.status.IncreReadAmount()
		rpcConn.Lock()
		pendingResponse := rpcConn.RemovePendingResponse(respHeader.Seq)
		rpcConn.Unlock()
		if respHeader.ReplyType == ReplyTypePong {
			continue
		}
		pendingResponse.err = respHeader.Error
		// @todo  call do not observes this pending response,ReadResponseBody use nil instead of pendingResponse.reply
		if respHeader.HaveReply() {
			if err = rpcConn.ReadResponseBody(pendingResponse.reply); err != nil {
				if isNetError(err) {
					pendingResponse.err = ErrNetReadFail.SetError(err)
					pendingResponse.done <- true
					break
				}
				pendingResponse.err = ErrGobParseErr.SetError(err)
			}
		}
		pendingResponse.done <- true
		cp.Lock()
		rpcConn.Lock()
		if rpcConn.netError == nil {
			rpcConn.lastUseTime = time.Now()
			rpcConn.callCount++
			if len(rpcConn.pendingResponses) == 0 {
				cp.MarkAsIdle(rpcConn)
			}
		}
		rpcConn.Unlock()
		cp.Unlock()
	}
	rpcConn.exitWriteNotify <- true // forbidden write request to this connection

	if rpcConn.isCloseByGCTimer() {
		err = ErrNetReadDeadlineArrive
	}
	rpcConn.Lock()
	rpcConn.netError = err
	rmap := rpcConn.ClearPendingResponses()
	rpcConn.Unlock()
	cp.Lock()
	cp.RemoveConn(rpcConn)
	cp.Unlock()
	rpcConn.Close()
	close(rpcConn.pendingRequests)
	for _, resp := range rmap {
		resp.err = ErrPendingWireBroken
		resp.done <- true
	}
}

// serve write connection
func (cp *ConnPool) serveWrite(rpcConn *ConnDriver) {
	var err error
	for {
		select {
		case request, ok := <-rpcConn.pendingRequests:
			if !ok {
				err = &Error{500, ErrTypeLogic, "client write channel close"}
				goto fail
			}
			if isPending := request.IsPending(); !isPending {
				rpcConn.Lock()
				rpcConn.RemovePendingResponse(request.header.Seq)
				rpcConn.Unlock()
				continue
			}
			// write request
			if err = rpcConn.SetWriteDeadline(time.Now().Add(request.writeTimeout)); err != nil {
				goto fail
			}
			if err = rpcConn.WriteRequestHeader(request.header); err != nil {
				// println("write request: ", err.Error())
				goto fail
			}
			if request.header.IsPing() {
				if err = rpcConn.FlushWriteToNet(); err != nil {
					goto fail
				}
				break
			}
			if err = rpcConn.WriteRequestBody(request.body); err != nil {
				goto fail
			}
			if err = rpcConn.FlushWriteToNet(); err != nil {
				goto fail
			}
		case <-rpcConn.exitWriteNotify:
			goto fail
		}
	}
fail:
	// close by gc overwrite the common net error
	if rpcConn.isCloseByGCTimer() {
		err = ErrNetTimerGCArrive
	}
	if err == nil {
		return
	}
	rpcConn.Lock()
	if rpcConn.netError == nil {
		rpcConn.netError = err
	}
	rpcConn.Unlock()
}

// GC the timer which is timeout
// review_deadlock cp.lock() -> conn.timeLock.lock()
func (cp *ConnPool) GCTimer() {
	connsTimeout := []*ConnDriver{}
	for {
		now := time.Now()
		cp.Lock()
		for e := cp.openConnsPool.workingList.Front(); e != nil; e = e.Next() {
			conn := e.Value.(*ConnDriver)
			conn.timeLock.RLock()
			if conn.readDeadline.Before(now) || conn.writeDeadline.Before(now) {
				connsTimeout = append(connsTimeout, conn)
			}
			conn.timeLock.RUnlock()
		}
		for _, conn := range connsTimeout {
			cp.openConnsPool.RemoveFromList(conn)
		}
		cp.Unlock()
		for _, conn := range connsTimeout {
			conn.timeLock.Lock()
			conn.closeByTimerGC = true
			conn.timeLock.Unlock()
			conn.Close()
		}
		connsTimeout = connsTimeout[0:0]
		time.Sleep(DefaultTimerGCInterval)
	}
}

func (cp *ConnPool) ServeIdlePing() {
	for {
		connPingSlice := []*ConnDriver{}
		connCloseSlice := []*ConnDriver{}
		idleCount := 0
		now := time.Now()
		cp.Lock()
		for e := cp.openConnsPool.idleList.Front(); e != nil; e = e.Next() {
			idleCount++
			if idleCount <= cp.maxIdleConns {
				connPingSlice = append(connPingSlice, e.Value.(*ConnDriver))
			} else {
				conn := e.Value.(*ConnDriver)
				connCloseSlice = append(connCloseSlice, conn)
			}
		}
		for _, conn := range connCloseSlice {
			cp.openConnsPool.RemoveFromList(conn)
		}
		cp.Unlock()
		go func(pingSlice, closeSlice []*ConnDriver) {
			for _, rpcConn := range pingSlice {
				rpcConn.Lock()
				interval := now.Sub(rpcConn.lastUseTime)
				rpcConn.Unlock()
				if interval > DefaultPingInterval &&
					interval < 2*DefaultServerIdleTimeout {
					cp.client.writePing(rpcConn)
					continue
				}
			}
			for _, rpcConn := range closeSlice {
				rpcConn.Lock()
				rpcConn.netError = ErrIdleClose
				rpcConn.Close()
				rpcConn.Unlock()
			}
		}(connPingSlice, connCloseSlice)
		time.Sleep(DefaultPingInterval)
	}

}

// conn opened pool
type OpensPool struct {
	workingList *list.List
	idleList    *list.List
}

func NewOpenPool() *OpensPool {
	return &OpensPool{
		workingList: list.New(),
		idleList:    list.New(),
	}
}

func (op *OpensPool) WorkingPushBack(conn *ConnDriver) {
	e := op.workingList.PushBack(conn)
	conn.workingElement = e
}

// move the front connection to back then return the back connection
func (op *OpensPool) WorkingMoveFrontToBack() (conn *ConnDriver) {
	e := op.workingList.Front()
	if e == nil {
		return nil
	}
	op.workingList.MoveToBack(e)
	e = op.workingList.Back()
	conn = e.Value.(*ConnDriver)
	conn.workingElement = e
	return
}

func (op *OpensPool) IdlePopFront() (conn *ConnDriver) {
	e := op.idleList.Front()
	if e == nil {
		return nil
	}
	conn = e.Value.(*ConnDriver)
	op.idleList.Remove(e)
	conn.idleElement = nil
	return
}

func (op *OpensPool) IdlePushBack(conn *ConnDriver) {
	if conn.idleElement == nil {
		e := op.idleList.PushBack(conn)
		conn.idleElement = e
	}

}

// remove the conn from working list and idle list
func (op *OpensPool) RemoveFromList(conn *ConnDriver) {
	if conn.workingElement != nil {
		op.workingList.Remove(conn.workingElement)
		conn.workingElement = nil

	}
	if conn.idleElement != nil {
		op.idleList.Remove(conn.idleElement)
		conn.idleElement = nil
	}
}

func (op *OpensPool) Len() int {
	return op.workingList.Len()
}
