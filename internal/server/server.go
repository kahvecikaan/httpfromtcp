package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	closed   atomic.Bool
}

func Serve(port int) (*Server, error) {
	addr := fmt.Sprintf(":%d", port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{listener: listener}

	go s.listen()

	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)      // mark as closed
	return s.listener.Close() // close the TCP listener
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				// expected during shutdown
				return
			}

			log.Println("Failed to accept connection:", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"Hello, World!"

	_, err := conn.Write([]byte(response))
	if err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
