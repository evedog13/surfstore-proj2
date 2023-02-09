package tritonhttp

import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Headers stores the key-value HTTP headers
	Headers map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

func MakeRequest(br *bufio.Reader) (req *Request, contentReceived bool, err error) {
	req = &Request{}

	firstLine, err := ReadLine(br)
	if err != nil {
		return nil, false, err // return false when reads the first line unsuccessfully
	}

	req.Method, req.URL, req.Proto, err = parseRequestFirstLine(firstLine)
	if err != nil {
		return nil, true, err
	}

	// check if the URL is malformed or invalid
	if req.URL[0] != '/' || req.Method != "GET" {
		return nil, true, err
	}

	req.Headers = make(map[string]string)

	for {
		line, err := ReadLine(br)
		key, value, err := parseRequestRestLine(line)
		req.Headers[key] = value // fill the headers
		if line == "" {
			break
		}
		if err != nil {
			return nil, true, err
		}
	}
	// req.Headers = headers // fill completely
	req.Host = req.Headers["Host"]
	if req.Headers["Connection"] == "close" {
		req.Close = true
	}
	return req, true, err
}

func parseRequestFirstLine(line string) (string, string, string, error) {
	fields := strings.SplitN(line, " ", 3) // split into 2 parts
	// check the length of my fields
	if len(fields) != 3 {
		return "", "", "", fmt.Errorf("Could not parse the request line")
	}

	return fields[0], fields[1], fields[2], nil
}

func parseRequestRestLine(line string) (string, string, error) {
	fields := strings.SplitN(line, ":", 2) // split into 2 parts
	// check the length of my fields
	if len(fields) != 2 {
		return "", "", fmt.Errorf("Could not parse the request line")
	}

	return CanonicalHeaderKey(strings.TrimSpace(fields[0])), strings.TrimSpace(fields[1]), nil
}

// ReadLine reads a single line ending with "\r\n" from br,
// striping the "\r\n" line end from the returned string.
// If any error occurs, data read before the error is also returned.
// You might find this function useful in parsing requests.
func ReadLine(br *bufio.Reader) (string, error) {
	var line string
	for {
		s, err := br.ReadString('\n')
		line += s
		// Return the error
		if err != nil {
			return line, err
		}
		// Return the line when reaching line end
		if strings.HasSuffix(line, "\r\n") {
			// Striping the line end
			line = line[:len(line)-2]
			return line, nil
		}
	}
}
