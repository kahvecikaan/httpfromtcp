package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	out := make(chan string)

	go func() {
		defer f.Close()
		defer close(out)

		currentLine := ""

		for {
			data := make([]byte, 8)
			n, err := f.Read(data)

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
	f, err := os.Open("messages.txt")
	if err != nil {
		log.Fatal("error", err)
	}

	lineCh := getLinesChannel(f)

	for line := range lineCh {
		fmt.Printf("read: %s\n", line)
	}
}
