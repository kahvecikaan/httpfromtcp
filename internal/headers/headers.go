package headers

import (
	"bytes"
	"fmt"
)

type Headers map[string]string

var CRLF = []byte("\r\n")

func NewHeaders() Headers {
	return map[string]string{}
}

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed header")
	}

	if bytes.HasSuffix(parts[0], []byte(" ")) {
		return "", "", fmt.Errorf("invalid spacing: space before colon")
	}

	name := string(bytes.TrimSpace(parts[0]))
	value := string(bytes.TrimSpace(parts[1]))

	return name, value, nil
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
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

	h[fieldName] = fieldValue
	
	return readIdx + len(CRLF), false, nil
}
