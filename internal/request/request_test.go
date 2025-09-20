package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLineParse(t *testing.T) {
	// Test: Good GET Request line
	r, err := RequestFromReader(strings.NewReader("GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good GET Request line with path
	r, err = RequestFromReader(strings.NewReader("GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good POST Request with path
	r, err = RequestFromReader(strings.NewReader("POST /api/users HTTP/1.1\r\nHost: localhost:42069\r\nContent-Type: application/json\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/api/users", r.RequestLine.RequestTarget)
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
