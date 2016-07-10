package gorpc

import (
	"reflect"
	"sync"
)

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

type RpcStatus struct{ server *Server }

func (inner *RpcStatus) CallStatus(detail bool, serverStatus *string) error {
	*serverStatus = inner.server.status.String()
	return nil
}
