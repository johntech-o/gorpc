// @todo while conn broke ,move the pending request to other -> done
// @todo call with retry setting -> done
// @todo send only call -> done
// @todo calculate Bandwidth consumption

package gorpc

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/johntech-o/timewheel"
)

const (
	CALL_RETRY_TIMES = 1
)

type NetOptions struct {
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

func NewNetOptions(connectTimeout, readTimeOut, writeTimeout time.Duration) *NetOptions {
	return &NetOptions{connectTimeout, readTimeOut, writeTimeout}
}

type ServerOptions struct {
	address      string
	maxOpenConns int
	maxIdleConns int
}

func NewServerOptions(serverAddress string, maxOpenConns, maxIdleConns int) *ServerOptions {
	return &ServerOptions{serverAddress, maxOpenConns, maxIdleConns}
}

type Client struct {
	sync.RWMutex                        // guard following
	cpMap          map[string]*ConnPool // connection pool map
	addressSlice   []string
	serviceOptions map[string]*NetOptions
	methodOptions  map[string]map[string]*NetOptions
	serverOptions  *NetOptions
}

func NewClient(netOptions *NetOptions) *Client {
	c := Client{
		cpMap:          make(map[string]*ConnPool),
		serverOptions:  netOptions,
		serviceOptions: make(map[string]*NetOptions),
		methodOptions:  make(map[string]map[string]*NetOptions),
	}
	return &c
}

func (this *Client) AddServers(servers []*ServerOptions) *ConnPool {
	this.Lock()
	var cp *ConnPool
	var ok bool
	for _, server := range servers {
		if cp, ok = this.cpMap[server.address]; ok {
			cp.Lock()
			cp.maxOpenConns = server.maxOpenConns
			cp.maxIdleConns = server.maxIdleConns
			cp.Unlock()
		} else {
			cp = NewConnPool(server.address, server.maxOpenConns, server.maxIdleConns)
			cp.client = this
			this.cpMap[server.address] = cp
			this.addressSlice = append(this.addressSlice, server.address)
		}
	}
	this.Unlock()
	return cp
}

func (this *Client) RemoveServers(addresses map[string]struct{}) {
	s := []string{}
	this.Lock()
	for address, _ := range addresses {
		delete(this.cpMap, address)
	}
	for _, address := range this.addressSlice {
		if _, ok := addresses[address]; ok {
			continue
		}
		s = append(s, address)
	}
	this.addressSlice = s
	this.Unlock()
}

func (this *Client) SetServerNetOptions(netOptions *NetOptions) error {
	this.Lock()
	this.serverOptions = netOptions
	this.Unlock()
	return nil
}

func (this *Client) SetServiceNetOptions(service string, netOptions *NetOptions) error {
	this.Lock()
	this.serviceOptions[service] = netOptions
	this.Unlock()
	return nil
}

func (this *Client) SetMethodNetOptinons(service, method string, netOptions *NetOptions) error {
	this.Lock()
	if _, ok := this.methodOptions[service]; !ok {
		this.methodOptions[service] = make(map[string]*NetOptions)
	}
	this.methodOptions[service][method] = netOptions
	this.Unlock()
	return nil
}

// set reply to nil means server send response immediately before execute service.method
func (this *Client) Call(service, method string, args interface{}, reply interface{}) *Error {
	serverAddress, err := this.getAddress()
	if err != nil {
		return err
	}
	return this.CallWithAddress(serverAddress, service, method, args, reply)
}

// set reply to nil means server send response immediately then exec  service.method
func (this *Client) CallWithAddress(serverAddress, service, method string, args interface{}, reply interface{}) *Error {
	if serverAddress == "" {
		return ErrInvalidAddress.SetReason("client remote address is empty")
	}
	var (
		err     *Error
		rpcConn *ConnDriver
		presp   *PendingResponse
		request *Request
	)
	connectTimeout, readTimeout, writeTimeout := this.getTimeout(service, method)
	this.RLock()
	cp, ok := this.cpMap[serverAddress]
	this.RUnlock()
	if !ok {
		cp = this.AddServers([]*ServerOptions{NewServerOptions(serverAddress, DefaultMaxOpenConns, DefaultMaxIdleConns)})
	}
	rpcConn, err = cp.Conn(connectTimeout, false)
	if err != nil {
		return err
	}
	// init request
	request = NewRequest()
	request.header.Service = service
	request.header.Method = method
	if reply == nil {
		request.header.CallType = RequestSendOnly
	}
	request.body = args
	request.writeTimeout = writeTimeout
	// init pending response
	presp = NewPendingResponse()
	presp.reply = reply
	retryTimes := 0
	timer := timewheel.NewTimer(readTimeout + writeTimeout)
Retry:
	if err = this.transfer(rpcConn, request, presp); err != nil {
		goto final
	}

	select {
	// overload of server will cause timeout
	case <-timer.C:
		request.freePending()
		err = ErrRequestTimeout
		goto final
	case <-presp.done:
		if presp.err == nil {
			goto final
		}
		err = presp.err
		presp.err = nil
		goto final
	}
final:
	// fmt.Println("client error: ", err)
	if err == nil {
		// can free request/presp object
		return nil
	}
	if err == ErrRequestTimeout {
		// can not free request/presp object left to gc
		return err
	}
	if !CanRetry(err) || retryTimes == CALL_RETRY_TIMES {
		// can free request/presp object left to gc
		return err
	}
	time.Sleep(time.Millisecond * 5)
	retryTimes++
	if rpcConn, err = cp.Conn(connectTimeout, true); err != nil {
		// can free request/presp object
		return err
	}
	goto Retry
	return nil
}

// connections status statistics
func (this *Client) ConnsStatus() string {
	status := make(map[string]*ClientStatus)
	this.RLock()
	for serverAddress, cp := range this.cpMap {
		status[serverAddress] = cp.poolStatus()
	}
	this.RUnlock()

	var connsStatus = struct {
		Result map[string]map[string]uint64
		Errno  int
	}{Result: make(map[string]map[string]uint64)}

	for address, s := range status {
		connsStatus.Result[address] = make(map[string]uint64)
		connsStatus.Result[address]["idle"] = s.idleAmount
		connsStatus.Result[address]["working"] = s.workingAmount
		connsStatus.Result[address]["creating"] = s.creatingAmount
		connsStatus.Result[address]["readAmount"] = s.readAmount
	}
	result, err := json.Marshal(connsStatus)
	if err != nil {
		return `{"Result":"inner error marshal error ` + err.Error() + `","Errno":500}`
	}
	return string(result)
}

// client qps statistics
func (this *Client) Qps() string {
	var qps = struct {
		Result uint64
		Errno  int
	}{}

	addressesReadAmount := make(map[string]uint64)
	this.RLock()
	for address, cp := range this.cpMap {
		addressesReadAmount[address] = cp.status.ReadAmount()
	}
	this.RUnlock()
	time.Sleep(time.Second)
	this.RLock()
	for address, cp := range this.cpMap {
		if readAmount, ok := addressesReadAmount[address]; ok {
			qps.Result += cp.status.ReadAmount() - readAmount
			continue
		}
		qps.Result += cp.status.ReadAmount()
	}
	this.RUnlock()
	result, err := json.Marshal(qps)
	if err != nil {
		return `{"Result":"inner error marshal error ` + err.Error() + `","Errno":500}`
	}
	return string(result)
}

func (this *Client) transfer(rpcConn *ConnDriver, request *Request, presp *PendingResponse) *Error {
	var (
		sequence uint64
		err      error
	)
	rpcConn.Lock()
	if rpcConn.netError != nil {
		err = rpcConn.netError
		rpcConn.Unlock()
		goto fail
	}
	sequence = rpcConn.Sequence()
	request.header.Seq = sequence
	// add request and wait response
	if err = rpcConn.AddPendingRequest(request); err != nil {
		rpcConn.Unlock()
		goto fail
	}
	presp.seq = sequence
	presp.connId = rpcConn.connId
	rpcConn.AddPendingResponse(presp)
	rpcConn.Unlock()
	return nil
fail:
	if e, ok := err.(*Error); ok {
		return e
	}
	return ErrUnknow.SetError(err)
}

func (this *Client) writePing(rpcConn *ConnDriver) error {
	// init request
	request := NewRequest()
	request.header.Service = "go"
	request.header.Method = "p"
	request.header.CallType = RequestSendOnly
	request.body = nil
	request.writeTimeout = time.Second * 5
	// init pending response
	presp := NewPendingResponse()
	return this.transfer(rpcConn, request, presp)
}

// get readTimeout writeTimeout
func (this *Client) getTimeout(service, method string) (time.Duration, time.Duration, time.Duration) {
	this.RLock()
	var netOption *NetOptions
	if netOption = this.methodOptions[service][method]; netOption != nil {
		this.RUnlock()
		// println("service method")
		return netOption.connectTimeout, netOption.readTimeout, netOption.writeTimeout
	}
	if netOption = this.serviceOptions[service]; netOption != nil {
		this.RUnlock()
		goto final
	}
	netOption = this.serverOptions
	this.RUnlock()
	if netOption == nil {
		panic("invalid netOption")
	}

final:
	this.Lock()
	if serviceNetOption, ok := this.methodOptions[service]; ok {
		serviceNetOption[method] = netOption
	} else {
		option := make(map[string]*NetOptions)
		option[method] = netOption
		this.methodOptions[service] = option
	}
	this.Unlock()
	return netOption.connectTimeout, netOption.readTimeout, netOption.writeTimeout
}

func (this *Client) getAddress() (string, *Error) {
	this.RLock()
	defer this.RUnlock()
	count := len(this.addressSlice)
	if count == 0 {
		return "", ErrInvalidAddress.SetReason("empty address")
	}
	return this.addressSlice[rand.Intn(count)], nil
}
