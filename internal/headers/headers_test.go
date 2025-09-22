package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParse(t *testing.T) {
	// Test: Valid single header
	t.Run("Valid single header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "localhost:42069", headers.Get("Host"))
		assert.Equal(t, 23, n) // "Host: localhost:42069\r\n" = 23 bytes
		assert.False(t, done)
	})

	// Test: Valid single header with extra whitespace
	t.Run("Valid single header with extra whitespace", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("   Host:   localhost:42069   \r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "localhost:42069", headers.Get("Host"))
		assert.Equal(t, 31, n)
		assert.False(t, done)
	})

	// Test: Case insensitive headers (NEW - Assignment Requirement)
	t.Run("Case insensitive headers", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Content-Length: 100\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)

		// Should be able to get with any case variation
		assert.Equal(t, "100", headers.Get("content-length"))
		assert.Equal(t, "100", headers.Get("Content-Length"))
		assert.Equal(t, "100", headers.Get("CONTENT-LENGTH"))
		assert.Equal(t, "100", headers.Get("CoNtEnT-lEnGtH"))
		assert.Equal(t, 21, n)
		assert.False(t, done)
	})

	// Test: Mixed case header names become lowercase (NEW - Assignment Requirement)
	t.Run("Mixed case header names", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("User-Agent: curl/7.81.0\r\n")
		_, done, err := headers.Parse(data)
		require.NoError(t, err)

		// Verify we can access with lowercase
		assert.Equal(t, "curl/7.81.0", headers.Get("user-agent"))
		// Verify mixed case input still works
		assert.Equal(t, "curl/7.81.0", headers.Get("User-Agent"))
		assert.False(t, done)
	})

	// Test: Invalid character in header name (NEW - Assignment Requirement)
	t.Run("Invalid character in header name", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("H©st: localhost:42069\r\n")
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character in field name")
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	// Test: Valid 2 headers with existing headers
	t.Run("Valid 2 headers with existing headers", func(t *testing.T) {
		headers := NewHeaders()
		headers.Set("Existing", "value")

		// Parse first header
		data1 := []byte("Host: localhost:42069\r\n")
		n1, done1, err1 := headers.Parse(data1)
		require.NoError(t, err1)
		assert.False(t, done1)
		assert.Equal(t, 23, n1)

		// Parse second header
		data2 := []byte("User-Agent: curl/7.81.0\r\n")
		n2, done2, err2 := headers.Parse(data2)
		require.NoError(t, err2)
		assert.False(t, done2)
		assert.Equal(t, 25, n2)

		// Verify all headers exist (case insensitive access)
		assert.Equal(t, "value", headers.Get("existing"))
		assert.Equal(t, "localhost:42069", headers.Get("host"))
		assert.Equal(t, "curl/7.81.0", headers.Get("user-agent"))
	})

	// Test: Valid done
	t.Run("Valid done", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("\r\n") // Empty line signals end of headers
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 2, n) // Consumed the \r\n
		assert.True(t, done)
		assert.Equal(t, "", headers.Get("anykey")) // No headers, returns empty string
	})

	// Test: Invalid spacing header
	t.Run("Invalid spacing header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host : localhost:42069\r\n") // Space before colon
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "space before colon")
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	// Test: Incomplete header (no CRLF)
	t.Run("Incomplete header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost") // No \r\n
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 0, n) // No bytes consumed
		assert.False(t, done)
		assert.Equal(t, "", headers.Get("Host")) // No header added
	})

	// Test: Malformed header (no colon)
	t.Run("Malformed header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("InvalidHeader\r\n")
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "malformed header")
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	// Test: Valid token characters (Additional coverage)
	t.Run("Valid token characters", func(t *testing.T) {
		headers := NewHeaders()
		// Test header name with all valid special characters
		data := []byte("X-Custom-Header_123.test~value: some-value\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "some-value", headers.Get("x-custom-header_123.test~value"))
		assert.Equal(t, 44, n)
		assert.False(t, done)
	})

	// Test: Invalid token characters (Additional coverage)
	t.Run("Invalid token characters", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     string
			expected string
		}{
			{"Space in name", "Invalid Header: value\r\n", " "},
			{"Parentheses", "Header(test): value\r\n", "("},
			{"Equals sign", "Header=test: value\r\n", "="},
			{"Comma", "Header,test: value\r\n", ","},
			{"Invalid character", "H©st: localhost:42069\r\n\r\n", "©"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				headers := NewHeaders()
				data := []byte(tc.data)
				n, done, err := headers.Parse(data)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid character in field name")
				assert.Equal(t, 0, n)
				assert.False(t, done)
			})
		}
	})

	// Test: Empty field name (Additional coverage)
	t.Run("Empty field name", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte(": value\r\n")
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name cannot be empty")
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})
}
