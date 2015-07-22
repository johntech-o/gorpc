package gorpc

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
