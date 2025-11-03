package response

import (
	"fmt"
	"io"
	"net/http"

	"github.com/kahvecikaan/httpfromtcp/internal/headers"
)

type StatusCode uint16

const (
	StatusOK         StatusCode = 200
	StatusBadRequest StatusCode = 400
	StatusInternal   StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	reasonPhrase := http.StatusText(int(statusCode))
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)

	_, err := w.Write([]byte(statusLine))
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-type", "text/plain")
	return *h
}

func WriteHeaders(w io.Writer, hdrs headers.Headers) error {
	var writeErr error

	hdrs.ForEach(func(key string, value string) {
		if writeErr != nil {
			return
		}

		headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
		_, err := w.Write([]byte(headerLine))
		if err != nil {
			writeErr = err
		}
	})

	if writeErr != nil {
		return writeErr
	}

	_, err := w.Write([]byte("\r\n"))
	return err
}
