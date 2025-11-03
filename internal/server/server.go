package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/kahvecikaan/httpfromtcp/internal/response"
)

type Server struct {
	listener net.Listener
	closed   atomic.Bool
}

func Serve(port uint16) (*Server, error) {
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

	err := response.WriteStatusLine(conn, response.StatusOK)
	if err != nil {
		log.Printf("Failed to write status line: %v", err)
		return
	}

	headers := response.GetDefaultHeaders(0) // 0 bytes body for now

	err = response.WriteHeaders(conn, headers)
	if err != nil {
		log.Printf("Failed to write header: %v", err)
		return
	}
}
