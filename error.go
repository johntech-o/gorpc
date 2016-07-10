package gorpc

import (
	"fmt"
)

const (
	ErrTypeCritical = 1 // 0001
	ErrTypeLogic    = 2 // 0010
	ErrTypeCanRetry = 4 // 0100
	ErrTypeNet      = 8 // 1000
)

// client error,error code < 400
var (
	ErrRequestTimeout     = &Error{100, ErrTypeLogic, "client request time out"}
	ErrNoIdleConn         = &Error{101, ErrTypeLogic, "client no dile connection"}
	ErrNoWorkingConn      = &Error{102, ErrTypeLogic, "client no working connection"}
	ErrCallConnectTimeout = &Error{103, ErrTypeLogic, "client no connect timeout"}
	ErrIdleClose          = &Error{104, ErrTypeLogic, "client idle pool full close"}
	ErrUnknow             = &Error{107, ErrTypeLogic, ""}
	// critical error unexpected error
	ErrGobParseErr    = &Error{106, ErrTypeCritical, ""}
	ErrInvalidAddress = &Error{108, ErrTypeCritical, "client invalid address"}

	ErrNetConnectFail = &Error{109, ErrTypeNet, ""}
	ErrNetReadFail    = &Error{110, ErrTypeNet, ""}
	// client can retry once after receiving following errors
	ErrPendingWireBroken  = &Error{111, ErrTypeCanRetry, ""}
	ErrPendingRequestFull = &Error{121, ErrTypeCanRetry, "client pending request full"}
)

// server error,error code >= 400
var (
	ErrNotFound = &Error{400, ErrTypeCritical, "server invalid service or method"}
)

// user defined error, error code > 10000

type Error struct {
	Code   int
	Type   int
	Reason string
}

func NewError(code, typ int, reason string) *Error {
	return &Error{code, typ, reason}
}

//params err can not be a nil of *Error
func (this *Error) SetError(err error) *Error {
	e := *this
	if err != nil {
		e.Reason = err.Error()
	}
	return &e
}

func (this *Error) SetReason(err string) *Error {
	e := *this
	e.Reason = err
	return &e
}

// gob error stop retry
// errer type is not ErrTypeCanRetry stop retry
func CanRetry(err *Error) bool {
	if err == nil {
		return false
	}
	if len(err.Reason) > 4 && err.Reason[0:4] == "gob:" {
		return false
	}
	return (err.Type & ErrTypeCanRetry) > 0
}

func (this Error) Error() string {
	return fmt.Sprintf("code: %d type: %d reason: %s", this.Code, this.Type, this.Reason)
}

func (this Error) Errno() int {
	return this.Code
}

func IsRpcError(err interface{}) bool {
	switch err.(type) {
	case *Error:
		return true
	case Error:
		return true
	default:
		return false
	}
}

func isNetError(err error) bool {
	e := err.Error()
	if len(e) >= 4 && e[0:4] == "gob:" {
		return false
	}
	return true
}
