package data

// import (
// 	"strconv"
// 	"sync/atomic"
// )

type TestABC struct {
	A, B, C string
}

type TestRpcInt struct {
	i int
}

func (r *TestRpcInt) Update(n int, res *int) error {
	r.i = n
	*res = r.i + 100
	return nil
}

const EchoContent = "hello echo struct"

var variable int32 = 0

func (r *TestRpcInt) EchoStruct(arg TestABC, res *string) error {
	*res = EchoContent
	return nil
}

//
