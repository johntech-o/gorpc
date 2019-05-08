package main

import (
	"flag"
	"github.com/johntech-o/gorpc"
	"github.com/johntech-o/gorpc/example/data"
)

var L string

func init() {
	flag.StringVar(&L, "l", "127.0.0.1:6668", "remote address:port")
	flag.Parse()
}

func main() {
	s := gorpc.NewServer(L)
	s.Register(new(data.TestRpcInt))
	s.Register(new(data.TestRpcABC))
	s.Serve()
	panic("server fail")
}
