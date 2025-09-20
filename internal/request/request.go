package request

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
}

var ErrorMalformedReqLine = fmt.Errorf("malformed request-line")
var ErrorInvalidMethod = fmt.Errorf("invalid method")
var ErrorUnsupportedHttpVer = fmt.Errorf("unsupported http version")
var ErrorInvalidHttpFormat = fmt.Errorf("invalid http version format")
var IncompleteStartLine = fmt.Errorf("incomplete start line")
var SEPARATOR = "\r\n"

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

func parseRequestLine(msg string) (*RequestLine, string, error) {
	idx := strings.Index(msg, SEPARATOR)
	if idx == -1 {
		return nil, msg, IncompleteStartLine
	}

	startLine := msg[:idx]
	restOfMsg := msg[idx+len(SEPARATOR):]

	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, restOfMsg, ErrorMalformedReqLine
	}

	if err := validateMethod(parts[0]); err != nil {
		return nil, restOfMsg, err
	}

	if err := validateHttpVersion(parts[2]); err != nil {
		return nil, restOfMsg, err
	}

	versionParts := strings.Split(parts[2], "/")

	rl := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   versionParts[1],
	}

	return rl, restOfMsg, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("unable to io.ReadAll"),
			err,
		)
	}

	str := string(data)
	rl, _, err := parseRequestLine(str)

	if err != nil {
		return nil, err
	}

	return &Request{
		RequestLine: *rl,
	}, nil
}
