package main

import (
	"time"
	"github.com/johntech-o/gorpc"
	"github.com/johntech-o/gorpc/example/data"
	"flag"
)

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
	err := client.CallWithAddress(A, "TestRpcABC", "EchoStruct", data.TestRpcABC{"aaa", "bbb", "ccc"}, &strReturn)
	if err != nil {
		println(err.Error(),err.Errno())
	}
	println(strReturn)


	var intReturn *int
	client.CallWithAddress(A,"TestRpcInt","Update",5,&intReturn)
	if err != nil {
		println(err.Error(),err.Errno())
	}
	println(*intReturn)

}
