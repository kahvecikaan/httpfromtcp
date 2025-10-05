package headers

import (
	"bytes"
	"fmt"
	"strings"
)

var CRLF = []byte("\r\n")

type Headers struct {
	headers map[string]string
}

func (h *Headers) Get(key string) string {
	return h.headers[strings.ToLower(key)]
}

func (h *Headers) Set(key, value string) {
	key = strings.ToLower(key)

	if v, ok := h.headers[key]; ok {
		h.headers[key] = fmt.Sprintf("%s, %s", v, value)
	} else {
		h.headers[key] = value
	}
}

func (h *Headers) ForEach(fn func(key, value string)) {
	for k, v := range h.headers {
		fn(k, v)
	}
}

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}

func isValidTokenChar(char byte) bool {
	// ALPHA (A-Z, a-z)
	if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
		return true
	}
	// DIGIT (0-9)
	if char >= '0' && char <= '9' {
		return true
	}
	// Special tchar characters
	switch char {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}

func validateFieldName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("field name cannot be empty")
	}

	for i := 0; i < len(name); i++ {
		if !isValidTokenChar(name[i]) {
			return fmt.Errorf("invalid character in field name: %c", name[i])
		}
	}
	return nil
}

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed header")
	}

	if bytes.HasSuffix(parts[0], []byte(" ")) {
		return "", "", fmt.Errorf("invalid spacing: space before colon")
	}

	name := strings.ToLower(string(bytes.TrimSpace(parts[0])))
	value := string(bytes.TrimSpace(parts[1]))

	if err := validateFieldName(name); err != nil {
		return "", "", err
	}

	return name, value, nil
}

func (h *Headers) Parse(data []byte) (n int, done bool, err error) {
	readIdx := bytes.Index(data, CRLF)

	if readIdx == 0 {
		readIdx += len(CRLF)
		return len(CRLF), true, nil
	}

	if readIdx == -1 {
		return 0, false, nil
	}

	fieldName, fieldValue, err := parseHeader(data[:readIdx])
	if err != nil {
		return 0, false, err
	}

	h.Set(fieldName, fieldValue)

	return readIdx + len(CRLF), false, nil
}
