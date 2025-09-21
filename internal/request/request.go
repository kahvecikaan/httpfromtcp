package request

import (
	"fmt"
	"io"
	"strings"
)

type ParserState int

const (
	stateInitialized ParserState = iota
	stateDone
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	state       ParserState
}

func NewRequest() *Request {
	return &Request{state: stateInitialized}
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		rl, bytesConsumed, err := parseRequestLine(string(data))
		if err != nil {
			return 0, err
		}

		if bytesConsumed == 0 {
			return 0, nil
		}

		r.RequestLine = *rl
		r.state = stateDone
		return bytesConsumed, nil
	case stateDone:
		return 0, fmt.Errorf("trying to read data in done state")
	default:
		return 0, fmt.Errorf("unknown parser state")
	}
}

var ErrorMalformedReqLine = fmt.Errorf("malformed request-line")
var ErrorInvalidMethod = fmt.Errorf("invalid method")
var ErrorUnsupportedHttpVer = fmt.Errorf("unsupported http version")
var ErrorInvalidHttpFormat = fmt.Errorf("invalid http version format")
var IncompleteStartLine = fmt.Errorf("incomplete start line")
var crlf = "\r\n"
var bufferSize = 1024

func validateMethod(method string) error {
	if len(method) == 0 {
		return ErrorInvalidMethod
	}

	for _, char := range method {
		// Allow uppercase letters and hyphens only
		// This covers all standard methods (GET, POST, etc.) and reasonable extensions (CUSTOM-METHOD)
		if !((char >= 'A' && char <= 'Z') || char == '-') {
			return ErrorInvalidMethod
		}
	}
	return nil
}

func validateHttpVersion(version string) error {
	parts := strings.Split(version, "/")
	if len(parts) != 2 {
		return ErrorInvalidHttpFormat
	}

	if parts[0] != "HTTP" {
		return ErrorInvalidHttpFormat
	}

	if parts[1] != "1.1" {
		return ErrorUnsupportedHttpVer
	}

	return nil
}

func parseRequestLine(msg string) (*RequestLine, int, error) {
	idx := strings.Index(msg, crlf)
	if idx == -1 {
		return nil, 0, nil
	}

	reqLine := msg[:idx]

	parts := strings.Split(reqLine, " ")
	if len(parts) != 3 {
		return nil, 0, ErrorMalformedReqLine
	}

	if err := validateMethod(parts[0]); err != nil {
		return nil, 0, err
	}

	if err := validateHttpVersion(parts[2]); err != nil {
		return nil, 0, err
	}

	versionParts := strings.Split(parts[2], "/")

	rl := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   versionParts[1],
	}

	return rl, idx + len(crlf), nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := NewRequest()
	buf := make([]byte, bufferSize)
	readToIdx := 0

	for req.state != stateDone {
		if readToIdx >= len(buf) {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}

		n, err := reader.Read(buf[readToIdx:])
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		readToIdx += n

		bytesConsumed, err := req.parse(buf[:readToIdx])
		if err != nil {
			return nil, err
		}

		if bytesConsumed > 0 {
			copy(buf, buf[bytesConsumed:readToIdx])
			readToIdx -= bytesConsumed
		}
	}

	return req, nil
}
