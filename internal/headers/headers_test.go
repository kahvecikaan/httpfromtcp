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
		assert.Equal(t, "localhost:42069", headers["Host"])
		assert.Equal(t, 23, n) // "Host: localhost:42069\r\n" = 23 bytes
		assert.False(t, done)
	})

	// Test: Valid single header with extra whitespace
	t.Run("Valid single header with extra whitespace", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("   Host:   localhost:42069   \r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "localhost:42069", headers["Host"])
		assert.Equal(t, 31, n)
		assert.False(t, done)
	})

	// Test: Valid 2 headers with existing headers
	t.Run("Valid 2 headers with existing headers", func(t *testing.T) {
		headers := NewHeaders()
		headers["Existing"] = "value" // Pre-existing header

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

		// Verify all headers exist
		assert.Equal(t, "value", headers["Existing"])
		assert.Equal(t, "localhost:42069", headers["Host"])
		assert.Equal(t, "curl/7.81.0", headers["User-Agent"])
	})

	// Test: Valid done
	t.Run("Valid done", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("\r\n") // Empty line signals end of headers
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 2, n) // Consumed the \r\n
		assert.True(t, done)
		assert.Len(t, headers, 0) // No headers added
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
		assert.Len(t, headers, 0) // No headers added
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
}
