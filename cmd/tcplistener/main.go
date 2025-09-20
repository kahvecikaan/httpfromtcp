package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func getLinesChannel(r io.ReadCloser) <-chan string {
	out := make(chan string)

	go func() {
		defer r.Close()
		defer close(out)

		currentLine := ""

		for {
			data := make([]byte, 8)
			n, err := r.Read(data)

			if err != nil && err != io.EOF {
				log.Fatal("error", err)
			}

			if n == 0 {
				break
			}

			chunk := string(data[:n])
			parts := strings.Split(chunk, "\n")

			for i := 0; i < len(parts)-1; i++ {
				completeLine := currentLine + parts[i]
				out <- completeLine
				currentLine = ""
			}
			currentLine += parts[len(parts)-1]

			if err == io.EOF {
				break
			}
		}

		if currentLine != "" {
			out <- currentLine
		}
	}()

	return out
}

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("error: ", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("error: ", err)
		}

		for line := range getLinesChannel(conn) {
			fmt.Printf("read: %s\n", line)
		}
	}
}
