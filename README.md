# gorpc
a golang rpc framework designed for Large-scale distributed system

## run a gorpc test
``
    go test -v github.com/johntech-o/gorpc
``

command above will show performance info at the end of test output
 
```
client conn status:  {"Result":{"127.0.0.1:6668":{"creating":0,"idle":0,"readAmount":907910,"working":30}},"Errno":0}
client conn Qps   :  {"Result":73011,"Errno":0}
HeapObjects: 2981940 - 21634 = 2960306
TotalAlloc: 789211080 - 50114848 = 739096232
PauseTotalMs: 1 - 0 = 1
NumGC: 12 - 1 = 11
server call status:  {"Result":{"Call/s":71929,"CallAmount":980847,"Err/s":0,"ErrorAmount":0,"ReadBytes":50014893,"ReadBytes/s":3668379,"WriteBytes":30399247,"WriteBytes/s":2229799},"Errno":0}
client conn status:  {"Result":{"127.0.0.1:6668":{"creating":0,"idle":30,"readAmount":1000008,"working":0}},"Errno":0}
client conn Qps   :  {"Result":19164,"Errno":0}
 10% calls consume less than 12 ms
 20% calls consume less than 13 ms
 30% calls consume less than 13 ms
 40% calls consume less than 14 ms
 50% calls consume less than 14 ms
 60% calls consume less than 14 ms
 70% calls consume less than 14 ms
 80% calls consume less than 15 ms
 90% calls consume less than 15 ms
100% calls consume less than 76 ms
request amount: 1000000, cost times : 14 second, average Qps: 69428
Max Client Qps: 73011
--- PASS: TestEchoStruct (15.44s)
	gorpc_test.go:240: TestEchoStruct result: hello echo struct ,count: 1000000
PASS
ok  	github.com/johntech-o/gorpc	15.481s

```

## example

run command on terminal to start the rpc server

`go run example/serverDemo/main.go`

run command on another terminal to see result: 

`go run example/clientDemo/main.go`

### rpc service and method define in data/data.go


```
// service TestRpcABC Has a method EchoStruct 
type TestRpcABC struct {
	A, B, C string
}

func (r *TestRpcABC) EchoStruct(arg TestRpcABC, res *string) error {
	*res = EchoContent
	return nil
}

// service TestRpcInt Has a method Update
type TestRpcInt struct {
	i int
}

func (r *TestRpcInt) Update(n int, res *int) error {
	r.i = n
	*res = r.i + 100
	return nil
}

```


### client example example/client.go
```
var client *gorpc.Client
var A string

func init() {
	netOptions := gorpc.NewNetOptions(time.Second*10, time.Second*20, time.Second*20)
	client = gorpc.NewClient(netOptions)
	flag.StringVar(&A, "a", "127.0.0.1:6668", "remote address:port")
	flag.Parse()
}

func main() {
	var strReturn string
	// call remote service TestRpcABC and method EchoStruct
	err := client.CallWithAddress(A, "TestRpcABC", "EchoStruct", data.TestRpcABC{"aaa", "bbb", "ccc"}, &strReturn)
	if err != nil {
		println(err.Error(),err.Errno())
	}
	println(strReturn)


	var intReturn *int
	// call remote service TestRpcInt and method Update
	client.CallWithAddress(A,"TestRpcInt","Update",5,&intReturn)
	if err != nil {
		println(err.Error(),err.Errno())
	}
	println(*intReturn)

}
```

### server example example/server.go

```
func main() {
	s := gorpc.NewServer(L)
	// register the services: TestRpcInt and TestRpcABC 
	s.Register(new(data.TestRpcInt))
	s.Register(new(data.TestRpcABC))
	s.Serve()
	panic("server fail")
}
```


