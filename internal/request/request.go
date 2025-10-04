package request

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/kahvecikaan/httpfromtcp/internal/headers"
)

type ParserState int

const (
	StateInitialized ParserState = iota
	StateHeaders
	StateDone
)

const (
	CRLF       = "\r\n"
	bufferSize = 1024
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	state       ParserState
}

var (
	ErrMalformedReqLine   = fmt.Errorf("malformed request-line")
	ErrInvalidMethod      = fmt.Errorf("invalid method")
	ErrUnsupportedHttpVer = fmt.Errorf("unsupported http version")
	ErrInvalidHttpFormat  = fmt.Errorf("invalid http version format")
	ErrParserDone         = fmt.Errorf("trying to read data in done state")
	ErrUnknownState       = fmt.Errorf("unknown parser state")
)

func NewRequest() *Request {
	return &Request{
		Headers: *headers.NewHeaders(),
		state:   StateInitialized,
	}
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case StateInitialized:
		rl, bytesConsumed, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}

		if bytesConsumed == 0 {
			return 0, nil
		}

		r.RequestLine = *rl
		r.state = StateHeaders
		return bytesConsumed, nil

	case StateHeaders:
		bytesConsumed, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}

		if done {
			r.state = StateDone
		}

		return bytesConsumed, nil

	case StateDone:
		return 0, ErrParserDone

	default:
		return 0, ErrUnknownState
	}
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0

	for r.state != StateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return totalBytesParsed, err
		}

		if n == 0 {
			// need more data
			break
		}

		totalBytesParsed += n
	}

	return totalBytesParsed, nil
}

func validateMethod(method string) error {
	if len(method) == 0 {
		return ErrInvalidMethod
	}

	for _, char := range method {
		if !((char >= 'A' && char <= 'Z') || char == '-') {
			return ErrInvalidMethod
		}
	}
	return nil
}

func validateHttpVersion(version string) error {
	parts := strings.Split(version, "/")
	if len(parts) != 2 {
		return ErrInvalidHttpFormat
	}

	if parts[0] != "HTTP" {
		return ErrInvalidHttpFormat
	}

	if parts[1] != "1.1" {
		return ErrUnsupportedHttpVer
	}

	return nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	crlfBytes := []byte(CRLF)
	idx := bytes.Index(data, crlfBytes)
	if idx == -1 {
		return nil, 0, nil
	}

	reqLineBytes := data[:idx]
	bytesConsumed := idx + len(crlfBytes)

	parts := bytes.Split(reqLineBytes, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrMalformedReqLine
	}

	method := string(parts[0])
	target := string(parts[1])
	version := string(parts[2])

	if err := validateMethod(method); err != nil {
		return nil, 0, err
	}

	if err := validateHttpVersion(version); err != nil {
		return nil, 0, err
	}

	versionParts := strings.Split(version, "/")

	rl := &RequestLine{
		Method:        method,
		RequestTarget: target,
		HttpVersion:   versionParts[1],
	}

	return rl, bytesConsumed, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := NewRequest()
	buf := make([]byte, bufferSize)
	readToIdx := 0

	for req.state != StateDone {
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
