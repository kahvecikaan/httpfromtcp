package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahvecikaan/httpfromtcp/internal/server"
)

const port = 42069

func main() {
	srv, err := server.Serve(port)
	if err != nil {
		log.Fatalf("Error starting srv: %v", err)
	}
	defer srv.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
