package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/johntech-o/gorpc"
	"github.com/johntech-o/gorpc/data"
	// "github.com/johntech-o/gorpc/pprof"
)

var L string

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.StringVar(&L, "l", "10.108.87.71:6668", "remote address:port")
	flag.Parse()
	go func() {
		for {
			// pprof.DoProcessStat()
			// pprof.DoMemoryStats(false)
			time.Sleep(time.Second * 1)
		}
	}()

}

func main() {

	s := gorpc.NewServer(L)
	s.Register(new(data.TestRpcInt))
	s.Serve()

	panic("server fail")
}
