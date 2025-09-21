package request

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per call
// it's useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	if endIndex > len(cr.data) {
		endIndex = len(cr.data)
	}
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	// Test: Good GET Request line
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good GET Request line with path
	reader = &chunkReader{
		data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Custom method with hyphen
	r, err = RequestFromReader(strings.NewReader("CUSTOM-METHOD /test HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "CUSTOM-METHOD", r.RequestLine.Method)

	// Test: PATCH method
	r, err = RequestFromReader(strings.NewReader("PATCH /users/123 HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "PATCH", r.RequestLine.Method)

	// Test: Invalid number of parts in request line
	_, err = RequestFromReader(strings.NewReader("/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.Error(t, err)

	// Test: Too many parts in request line
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTP/1.1 EXTRA\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Invalid method (lowercase)
	_, err = RequestFromReader(strings.NewReader("get /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Invalid method (contains numbers)
	_, err = RequestFromReader(strings.NewReader("GET123 /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Invalid method (contains special characters)
	_, err = RequestFromReader(strings.NewReader("GET$ /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Empty method
	_, err = RequestFromReader(strings.NewReader(" /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Invalid HTTP version (HTTP/2.0)
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTP/2.0\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Invalid HTTP version (HTTP/1.0)
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTP/1.0\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Malformed HTTP version (no slash)
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTP1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Malformed HTTP version (wrong protocol)
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTPS/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Missing \r\n separator
	_, err = RequestFromReader(strings.NewReader("GET /coffee HTTP/1.1Host: localhost:42069\r\n\r\n"))
	require.Error(t, err)

	// Test: Complex request target with query params
	r, err = RequestFromReader(strings.NewReader("GET /search?q=test&limit=10 HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "/search?q=test&limit=10", r.RequestLine.RequestTarget)
}

func TestRequestLineParseStreaming(t *testing.T) {
	baseRequest := "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"

	// Test various chunk sizes for the same request
	chunkSizes := []int{1, 2, 3, 5, 8, 16, 50, 100}

	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(t *testing.T) {
			reader := &chunkReader{
				data:            baseRequest,
				numBytesPerRead: chunkSize,
			}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, "GET", r.RequestLine.Method)
			assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
			assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
		})
	}

	// Test: Split exactly at \r\n boundary
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
		numBytesPerRead: 15, // This should split right at the \r
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "GET", r.RequestLine.Method)

	// Test: Custom method with various chunk sizes
	customRequest := "CUSTOM-METHOD /test HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"
	for _, chunkSize := range []int{1, 7, 13} {
		reader := &chunkReader{
			data:            customRequest,
			numBytesPerRead: chunkSize,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "CUSTOM-METHOD", r.RequestLine.Method)
	}

	// Test: Error cases with chunked reading
	errorCases := []struct {
		name string
		data string
	}{
		{"Invalid method lowercase", "get /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"},
		{"Invalid method numbers", "GET123 /coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"},
		{"Invalid HTTP version", "GET /coffee HTTP/2.0\r\nHost: localhost:42069\r\n\r\n"},
		{"Malformed request line", "/coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &chunkReader{
				data:            tc.data,
				numBytesPerRead: 2, // Small chunks to stress test
			}
			_, err := RequestFromReader(reader)
			require.Error(t, err)
		})
	}

	// Test: Complex request target with query params
	complexRequest := "GET /search?q=test&limit=10 HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"
	reader = &chunkReader{
		data:            complexRequest,
		numBytesPerRead: 4,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "/search?q=test&limit=10", r.RequestLine.RequestTarget)
}
