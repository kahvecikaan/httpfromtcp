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

func TestParseHeaders(t *testing.T) {
	// Test: Standard Headers
	t.Run("Standard Headers", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			numBytesPerRead: 3,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "localhost:42069", r.Headers.Get("host"))
		assert.Equal(t, "curl/7.81.0", r.Headers.Get("user-agent"))
		assert.Equal(t, "*/*", r.Headers.Get("accept"))
	})

	// Test: Empty Headers
	t.Run("Empty Headers", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\n\r\n",
			numBytesPerRead: 5,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		// Verify request line was parsed
		assert.Equal(t, "GET", r.RequestLine.Method)
		assert.Equal(t, "/", r.RequestLine.RequestTarget)
		// Verify no headers exist
		assert.Equal(t, "", r.Headers.Get("host"))
		assert.Equal(t, "", r.Headers.Get("content-length"))
	})

	// Test: Malformed Header
	t.Run("Malformed Header", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nHostWithNoColonAtAll\r\n\r\n",
			numBytesPerRead: 3,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "malformed header")
	})

	// Test: Duplicate Headers
	t.Run("Duplicate Headers", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nSet-Cookie: session=abc\r\nSet-Cookie: user=xyz\r\nSet-Cookie: theme=dark\r\n\r\n",
			numBytesPerRead: 10,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		// Should combine all values with comma separation
		assert.Equal(t, "session=abc, user=xyz, theme=dark", r.Headers.Get("set-cookie"))
	})

	// Test: Case Insensitive Headers
	t.Run("Case Insensitive Headers", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nContent-Length: 100\r\nContent-Type: application/json\r\n\r\n",
			numBytesPerRead: 7,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		// Should be accessible with any case variation
		assert.Equal(t, "100", r.Headers.Get("content-length"))
		assert.Equal(t, "100", r.Headers.Get("Content-Length"))
		assert.Equal(t, "100", r.Headers.Get("CONTENT-LENGTH"))
		assert.Equal(t, "application/json", r.Headers.Get("content-type"))
		assert.Equal(t, "application/json", r.Headers.Get("Content-Type"))
	})

	// Test: Missing End of Headers
	t.Run("Missing End of Headers", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nHost: localhost\r\nUser-Agent: curl\r\n",
			numBytesPerRead: 5,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		// Parser should parse what it can without erroring
		assert.Equal(t, "GET", r.RequestLine.Method)
		assert.Equal(t, "localhost", r.Headers.Get("host"))
		assert.Equal(t, "curl", r.Headers.Get("user-agent"))
		// State should not be done since we never got the empty line
	})

	// Test: Headers with various chunk sizes
	t.Run("Headers with various chunk sizes", func(t *testing.T) {
		data := "POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: 42\r\n\r\n"
		chunkSizes := []int{1, 3, 7, 15, 50}

		for _, chunkSize := range chunkSizes {
			t.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(t *testing.T) {
				reader := &chunkReader{
					data:            data,
					numBytesPerRead: chunkSize,
				}
				r, err := RequestFromReader(reader)
				require.NoError(t, err)
				assert.Equal(t, "POST", r.RequestLine.Method)
				assert.Equal(t, "example.com", r.Headers.Get("host"))
				assert.Equal(t, "application/json", r.Headers.Get("content-type"))
				assert.Equal(t, "42", r.Headers.Get("content-length"))
			})
		}
	})

	// Test: Header with invalid character
	t.Run("Header with invalid character", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nH©st: localhost\r\n\r\n",
			numBytesPerRead: 10,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character in field name")
	})

	// Test: Header with space before colon
	t.Run("Header with space before colon", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET / HTTP/1.1\r\nHost : localhost\r\n\r\n",
			numBytesPerRead: 8,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "space before colon")
	})

	// Test: Many headers
	t.Run("Many headers", func(t *testing.T) {
		reader := &chunkReader{
			data: "GET /api HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"User-Agent: test-client/1.0\r\n" +
				"Accept: application/json\r\n" +
				"Accept-Encoding: gzip, deflate\r\n" +
				"Accept-Language: en-US\r\n" +
				"Authorization: Bearer token123\r\n" +
				"Cache-Control: no-cache\r\n" +
				"Connection: keep-alive\r\n" +
				"\r\n",
			numBytesPerRead: 20,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "example.com", r.Headers.Get("host"))
		assert.Equal(t, "test-client/1.0", r.Headers.Get("user-agent"))
		assert.Equal(t, "application/json", r.Headers.Get("accept"))
		assert.Equal(t, "gzip, deflate", r.Headers.Get("accept-encoding"))
		assert.Equal(t, "en-US", r.Headers.Get("accept-language"))
		assert.Equal(t, "Bearer token123", r.Headers.Get("authorization"))
		assert.Equal(t, "no-cache", r.Headers.Get("cache-control"))
		assert.Equal(t, "keep-alive", r.Headers.Get("connection"))
	})
}

func TestBodyParsing(t *testing.T) {
	// Test: Standard Body
	t.Run("Standard Body", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 13\r\n" +
				"\r\n" +
				"hello world!\n",
			numBytesPerRead: 3,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "hello world!\n", string(r.Body))
	})

	// Test: Empty Body, 0 reported content length
	t.Run("Empty Body, 0 reported content length", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			numBytesPerRead: 5,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, "", string(r.Body))
		assert.Len(t, r.Body, 0)
	})

	// Test: Empty Body, no reported content length
	t.Run("Empty Body, no reported content length", func(t *testing.T) {
		reader := &chunkReader{
			data: "GET /page HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			numBytesPerRead: 10,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Nil(t, r.Body)
	})

	// Test: Body shorter than reported content length
	t.Run("Body shorter than reported content length", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 20\r\n" +
				"\r\n" +
				"partial content",
			numBytesPerRead: 3,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err) // Should not error, just incomplete
		require.NotNil(t, r)
		// Body will contain what was received, but state won't be Done
		assert.Equal(t, "partial content", string(r.Body))
	})

	// Test: No Content-Length but Body Exists
	t.Run("No Content-Length but Body Exists", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n" +
				"unexpected body data",
			numBytesPerRead: 10,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		// Since no Content-Length, we assume no body
		assert.Nil(t, r.Body)
	})

	// Test: Body with various chunk sizes
	t.Run("Body with various chunk sizes", func(t *testing.T) {
		data := "POST /api HTTP/1.1\r\n" +
			"Content-Length: 25\r\n" +
			"\r\n" +
			`{"message":"hello world"}`

		chunkSizes := []int{1, 3, 7, 15, 50}

		for _, chunkSize := range chunkSizes {
			t.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(t *testing.T) {
				reader := &chunkReader{
					data:            data,
					numBytesPerRead: chunkSize,
				}
				r, err := RequestFromReader(reader)
				require.NoError(t, err)
				assert.Equal(t, `{"message":"hello world"}`, string(r.Body))
			})
		}
	})

	// Test: Large body
	t.Run("Large body", func(t *testing.T) {
		bodyContent := strings.Repeat("a", 1000)
		reader := &chunkReader{
			data: "POST /upload HTTP/1.1\r\n" +
				"Content-Length: 1000\r\n" +
				"\r\n" +
				bodyContent,
			numBytesPerRead: 50,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		assert.Equal(t, bodyContent, string(r.Body))
		assert.Len(t, r.Body, 1000)
	})

	// Test: Invalid Content-Length (non-numeric)
	t.Run("Invalid Content-Length non-numeric", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Content-Length: not-a-number\r\n" +
				"\r\n",
			numBytesPerRead: 10,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid content-length")
	})

	// Test: Negative Content-Length
	t.Run("Negative Content-Length", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Content-Length: -100\r\n" +
				"\r\n",
			numBytesPerRead: 10,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative value")
	})

	// Test: Content-Length too large
	t.Run("Content-Length exceeds maximum", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Content-Length: 999999999999\r\n" +
				"\r\n",
			numBytesPerRead: 10,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	// Test: Multiple Content-Length headers
	t.Run("Multiple Content-Length headers", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Content-Length: 10\r\n" +
				"Content-Length: 20\r\n" +
				"\r\n",
			numBytesPerRead: 10,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple content-length")
	})
}
