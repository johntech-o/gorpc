package gorpc

import (
	"errors"
	"sync"
	// "fmt"
	"log"
	"net"
	"reflect"
	"time"
	"unicode"
	"unicode/utf8"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

const (
	GobCodec    = 1
	CmdTypePing = "ping"
	CmdTypeErr  = "err"
	CmdTypeAck  = "ack"
)

const (
	TimerPoolSize = 10
)

type connsMap struct {
	conns map[ConnId]*ConnDriver
	sync.RWMutex
}

type TimerPool [TimerPoolSize]*connsMap

func NewTimerPool() *TimerPool {
	tp := &TimerPool{}
	for index, _ := range tp {
		tp[index] = &connsMap{conns: make(map[ConnId]*ConnDriver)}
	}
	return tp
}

func (tp *TimerPool) AddConn(conn *ConnDriver) {
	index := conn.connId % TimerPoolSize
	tp[index].Lock()
	tp[index].conns[conn.connId] = conn
	tp[index].Unlock()

}

func (tp *TimerPool) RemoveConn(conn *ConnDriver) {
	index := conn.connId % TimerPoolSize
	tp[index].Lock()
	delete(tp[index].conns, conn.connId)
	tp[index].Unlock()

}

type Server struct {
	serviceMap map[string]*service
	listener   *net.TCPListener
	codec      int
	status     *ServerStatus
	timerPool  *TimerPool
}

func NewServer(Address string) *Server {
	addr, err := net.ResolveTCPAddr("tcp", Address)
	if err != nil {
		panic("NewServer error:" + err.Error())
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic("NewServer error:" + err.Error())
	}
	s := &Server{
		serviceMap: make(map[string]*service),
		listener:   listener,
		codec:      GobCodec,
		status:     &ServerStatus{},
		timerPool:  NewTimerPool(),
	}
	s.Register(&RpcStatus{s})
	go s.serveTimerManage()
	return s
}

func (server *Server) Serve() {
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			//log.Print("Serv:", err.Error())
			continue
		}
		connTcp := conn.(*net.TCPConn)
		go server.serveConn(connTcp)
	}
}

func (server *Server) Close() error {
	return server.listener.Close()
}

// serve  read write deadline-timer of conn
func (server *Server) serveConn(conn *net.TCPConn) {
	rpcConn := NewConnDriver(conn, server)
	server.timerPool.AddConn(rpcConn)
	server.ServeLoop(rpcConn)
	server.timerPool.RemoveConn(rpcConn)
}

// dead-lock review timerPool.lock -> conn.timeLock
func (server *Server) serveTimerManage() {
	for i := 0; i < TimerPoolSize; i++ {
		go func(pool *connsMap) {
			connsTimeout := make([]*ConnDriver, 100)[0:0]
			for {
				now := time.Now()
				pool.Lock()
				for _, conn := range pool.conns {
					conn.timeLock.RLock()
					if conn.readDeadline.Before(now) || conn.writeDeadline.Before(now) {
						connsTimeout = append(connsTimeout, conn)
					}
					conn.timeLock.RUnlock()
				}
				pool.Unlock()
				time.Sleep(DefaultServerTimerGCInterval)
			}

		}(server.timerPool[i])
	}

}

// serve connection, read request and call service
func (server *Server) ServeLoop(conn *ConnDriver) {
	var err error
	var methodType *methodType
	var service *service
	for {
		reqHeader := NewRequestHeader()
		if err = conn.SetReadDeadline(time.Now().Add(DefaultServerIdleTimeout)); err != nil {
			goto fail
		}
		err = conn.ReadRequestHeader(reqHeader)
		if err != nil {
			goto fail
		}
		server.status.IncrCallAmount()
		if reqHeader.IsPing() {
			server.replyCmd(conn, reqHeader.Seq, nil, CmdTypePing)
			continue
		}
		// verify service and method
		service = server.serviceMap[reqHeader.Service]
		if service != nil {
			methodType = service.method[reqHeader.Method]
		}
		if service == nil || methodType == nil {
			err = conn.ReadRequestBody(nil)
			if err != nil {
				goto fail
			}
			server.replyCmd(conn, reqHeader.Seq, ErrNotFound, CmdTypeErr)
			continue
		}

		var argv, replyv reflect.Value
		// Decode the argument value.
		argIsValue := false // if true, need to indirect before calling.
		if methodType.ArgType.Kind() == reflect.Ptr {
			argv = reflect.New(methodType.ArgType.Elem())
		} else {
			argv = reflect.New(methodType.ArgType)
			argIsValue = true
		}
		err = conn.ReadRequestBody(argv.Interface())
		if err != nil {
			if isNetError(err) {
				log.Println("read request body with net error:", err, "method:", reqHeader.Method)
				goto fail
			} else {
				server.replyCmd(conn, reqHeader.Seq, &Error{400, ErrTypeCritical, err.Error()}, CmdTypeErr)
				continue
			}
		}
		if argIsValue {
			argv = argv.Elem()
		}
		replyv = reflect.New(methodType.ReplyType.Elem())
		if reqHeader.CallType == RequestSendOnly {
			go server.asyncCallService(conn, reqHeader.Seq, service, methodType, argv, replyv)
			continue
		}
		go server.callService(conn, reqHeader.Seq, service, methodType, argv, replyv)
	}
fail:
	server.status.IncrErrorAmount()
	conn.Lock()
	conn.netError = err
	conn.Unlock()
	conn.Close()
	return
}

func (server *Server) Status() *ServerStatusPerSecond {
	return server.status.Status()
}

func (server *Server) replyCmd(conn *ConnDriver, seq uint64, serverErr *Error, cmd string) {

	respHeader := NewResponseHeader()
	respHeader.Seq = seq
	switch cmd {
	case CmdTypePing:
		respHeader.ReplyType = ReplyTypePong
	case CmdTypeAck:
		respHeader.ReplyType = ReplyTypeAck
	case CmdTypeErr:
		respHeader.ReplyType = ReplyTypeAck
		respHeader.Error = serverErr
		// fmt.Println("replycmd send respHeader type error")
	}
	conn.Lock()
	server.SendFrame(conn, respHeader, reflect.ValueOf(nil))
	conn.Unlock()
	return
}

// send response first telling client that server has received the request,then execute the service
func (server *Server) asyncCallService(conn *ConnDriver, seq uint64, service *service, methodType *methodType, argv, replyv reflect.Value) {
	server.replyCmd(conn, seq, nil, CmdTypeAck)
	function := methodType.method.Func
	// Invoke the method, providing a new value for the reply.
	function.Call([]reflect.Value{service.rcvr, argv, replyv})
	return
}

// do service and send response to client
func (server *Server) callService(conn *ConnDriver, seq uint64, service *service, methodType *methodType, argv, replyv reflect.Value) {
	function := methodType.method.Func

	returnValues := function.Call([]reflect.Value{service.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()

	respHeader := NewResponseHeader()
	respHeader.ReplyType = ReplyTypeData
	respHeader.Seq = seq
	if errInter != nil {
		switch errInter.(type) {
		case *Error:
			server.replyCmd(conn, seq, errInter.(*Error), CmdTypeErr)
		case Error:
			e := errInter.(Error)
			server.replyCmd(conn, seq, &e, CmdTypeErr)
		case error:
			server.replyCmd(conn, seq, &Error{500, ErrTypeLogic, errInter.(error).Error()}, CmdTypeErr)
		}
		return
	}
	conn.Lock()
	err := server.SendFrame(conn, respHeader, replyv)
	conn.Unlock()
	if err != nil && !isNetError(err) {
		log.Fatalln("encoding error:" + err.Error())
	}
	return
}

func (server *Server) SendFrame(conn *ConnDriver, respHeader *ResponseHeader, replyv reflect.Value) error {
	var err error
	if conn.netError != nil {
		return conn.netError
	}
	err = conn.SetWriteDeadline(time.Now().Add(DefaultServerIdleTimeout))
	if err != nil {
		goto final
	}
	err = conn.WriteResponseHeader(respHeader)
	if err != nil {
		goto final
	}
	if respHeader.HaveReply() {
		err = conn.WriteResponseBody(replyv.Interface())
		if err != nil {
			goto final
		}
	}
	err = conn.FlushWriteToNet()
	if err != nil {
		goto final
	}
final:
	if err != nil {
		conn.netError = err
	}
	return err
}

func (server *Server) Register(rcvr interface{}) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !isExported(sname) {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	if _, present := server.serviceMap[sname]; present {
		return errors.New("rpc: service already defined: " + sname)
	}
	s.name = sname
	// Install the methods
	s.method = suitableMethods(s.typ, true)
	if len(s.method) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}
	server.serviceMap[s.name] = s
	return nil
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			if reportErr {
				log.Println("method", mname, "has wrong number of ins:", mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Println(mname, "argument type not exported:", argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Println("method", mname, "reply type not a pointer:", replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Println("method", mname, "reply type not exported:", replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Println("method", mname, "has wrong number of outs:", mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Println("method", mname, "returns", returnType.String(), "not error")
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

// Is this an exported - upper case - name
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// require err != nil
