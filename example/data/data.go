package data

const EchoContent = "hello echo struct"

var variable int32 = 0

type TestRpcABC struct {
	A, B, C string
}

func (r *TestRpcABC) EchoStruct(arg TestRpcABC, res *string) error {
	*res = EchoContent
	return nil
}


type TestRpcInt struct {
	i int
}

func (r *TestRpcInt) Update(n int, res *int) error {
	r.i = n
	*res = r.i + 100
	return nil
}




//
