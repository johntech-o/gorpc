// 测试写阻塞情况下的现象 2.6.32-220.4.2.el6.x86_64 系统参数为为默认情况
// linux下，client写入，最大packet大小在1024~16384，服务端最后确认的包大小65536~65793以内，可以认为是64k，有点零头不看源码估计是不知道原因了..
// 客户端写入大小为65536左右的时候阻塞，此后每6s发送一个空包，服务端返回ack包，但win大小为0，120s时候结束整个过程
// 服务端会返回rst包，client端返回connection reset by peer
// 写入大小是1k的时候，传输过程中，packet大小都有1024~16384情况，但如果一次写入出现16384后会递减回每次传输1024。写入大小是400k时候，每次都会写入16384的包
// 设置setWriteBuffer时候，可以写入更多数据，但实际上是缓存在客户端的，有个问题是，你设置大小和缓存是两倍关系，设置64k，实际可以写入的是 64k+64k*2
// 设置为32k，写入的大小为64k + 32k*2 其中固定的64k是对方已经确认的大小，应该已经在缓冲区去除了. 但这个值最小是12k，也就是你设置为小于12k的数字，比如设置为0，实际上
// 没有效果，最终写入大小大概是 64k + 12k * 2 = 88k 在我的服务器上设置为大于12k以上会增加缓冲效果
// 以上测试都是使用默认的服务端readbuffer情况
// 如果设置了服务端的readbuffer size，抓包看服务端buffer缓存了更多数据，但这个数值和设置大小也是略有出入,如果不设置默认是64KB，设置为42K服务端会缓存64K，
// 设置超过128KB，服务端缓冲大概是242K不变

package main

import (
	"crypto/rand"
	"fmt"
	"net"
	"runtime"
	"time"
)

const (
	KB             = 1024
	WRITE_BLOCK    = 16 * KB
	WRITE_BUF_SIZE = 32 * KB
	READ_BUF_SIZE  = 128 * KB
)

const (
	SERVER_ADDR = "127.0.0.1:10240"
)

func init() {
	fmt.Println("cpu num is: ", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	clientStartNotify := make(chan struct{})
	go server(clientStartNotify)
	<-clientStartNotify
	go client()
	select {}
}

func client() {
	addr, err := net.ResolveTCPAddr("tcp", SERVER_ADDR)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	conn.SetWriteBuffer(WRITE_BUF_SIZE)
	if err != nil {
		panic(err)
	}
	b := make([]byte, WRITE_BLOCK)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	amount := 0
	for i := 1; ; i++ {
		n, e := conn.Write(b)
		if e != nil {
			fmt.Println("client write error: ", e)
			break
		}
		amount += n
		fmt.Println("write amount KB: ", amount/KB, amount)
		// time.Sleep(time.Millisecond)
	}
	fmt.Println("client exit")
}

func server(notify chan struct{}) {
	addr, _ := net.ResolveTCPAddr("tcp", SERVER_ADDR)
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
		// handle error
	}
	notify <- struct{}{}
	for {
		conn, err := ln.AcceptTCP()
		conn.SetReadBuffer(0)
		if err != nil {
			fmt.Println("accept error: ", err)
		}
		go func(conn *net.TCPConn) {
			time.Sleep(time.Second * 5)
			conn.SetReadBuffer(READ_BUF_SIZE)
			fmt.Println(conn.LocalAddr())
		}(conn)
	}
}
