package test

import (
	"fmt"
	"net"
)

type Server struct {
	serveConn ServeConn
	listener  net.Listener
}

func NewServer(serveConn ServeConn) *Server {
	return &Server{
		serveConn: serveConn,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return err
	}
	s.listener = listener
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go s.serveConn.ServeConn(conn)
	}()
	return nil
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}
