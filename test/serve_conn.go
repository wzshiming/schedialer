package test

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type ServeConn interface {
	ServeConn(conn net.Conn)
}

type unacceptedServeConn struct{}

func UnacceptedServeConn() ServeConn {
	return unacceptedServeConn{}
}

func (unacceptedServeConn) ServeConn(conn net.Conn) {}

type delayServeConn struct {
	delay time.Duration
}

func DelayServeConn(delay time.Duration) ServeConn {
	return &delayServeConn{
		delay: delay,
	}
}

func (d *delayServeConn) ServeConn(conn net.Conn) {
	time.Sleep(d.delay)
}

type disconnectServeConn struct{}

func DisconnectServeConn() ServeConn {
	return disconnectServeConn{}
}

func (disconnectServeConn) ServeConn(conn net.Conn) {
	conn.Close()
}

type replyServeConn struct {
	code int
	body string
}

func ReplyServeConn(code int, body string) ServeConn {
	if code == 0 {
		code = http.StatusOK
	}
	if body == "" {
		body = http.StatusText(code)
	}
	return replyServeConn{
		code: code,
		body: body,
	}
}

func (r replyServeConn) ServeConn(conn net.Conn) {
	var response = fmt.Sprintf("HTTP/1.0 %d %s\r\n\r\n%s", r.code, http.StatusText(r.code), r.body)
	conn.Write([]byte(response))
}

type switchServeConn struct {
	indexFunc  func(size int) int
	serveConns []ServeConn
}

func SwitchServeConn(indexFunc func(size int) int, serveConns ...ServeConn) ServeConn {
	return &switchServeConn{
		indexFunc:  indexFunc,
		serveConns: serveConns,
	}
}

func (s *switchServeConn) ServeConn(conn net.Conn) {
	s.serveConns[s.indexFunc(len(s.serveConns))].ServeConn(conn)
}

func RandomSwitchServeConn(serveConns ...ServeConn) ServeConn {
	return SwitchServeConn(func(size int) int {
		return rand.Int() % size
	}, serveConns...)
}

func IntervalSwitchServeConn(interval time.Duration, serveConns ...ServeConn) ServeConn {
	start := time.Now()
	return SwitchServeConn(func(size int) int {
		return int(time.Since(start)/interval) % size
	}, serveConns...)
}

func CountSwitchServeConn(interval uint64, serveConns ...ServeConn) ServeConn {
	var count uint64
	return SwitchServeConn(func(size int) int {
		defer atomic.AddUint64(&count, 1)
		return int(count/interval) % size
	}, serveConns...)
}

type jointServeConn []ServeConn

func JointServeConn(serveConns ...ServeConn) ServeConn {
	if len(serveConns) == 1 {
		return serveConns[0]
	}
	return jointServeConn(serveConns)
}

func (j jointServeConn) ServeConn(conn net.Conn) {
	for _, c := range j {
		c.ServeConn(conn)
	}
}
