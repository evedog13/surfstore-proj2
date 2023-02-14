// package tritonhttp

// import (
// 	"bufio"
// 	"fmt"
// 	"strings"
// )

// type Request struct {
// 	Method string // e.g. "GET"
// 	URL    string // e.g. "/path/to/a/file"
// 	Proto  string // e.g. "HTTP/1.1"

// 	// Headers stores the key-value HTTP headers
// 	Headers map[string]string

// 	Host  string // determine from the "Host" header
// 	Close bool   // determine from the "Connection" header
// }

// func MakeRequest(br *bufio.Reader) (req *Request, contentReceived bool, err error) {
// 	req = &Request{}

// 	firstLine, err := ReadLine(br)
// 	// fmt.Print("firstLine: ")
// 	// fmt.Println(firstLine)
// 	// fmt.Print("err: ")
// 	// fmt.Println(err)
// 	if err != nil {
// 		// fmt.Print("readline->makerequest: ")
// 		fmt.Println(err)
// 		return nil, false, err // return false when reads the first line unsuccessfully
// 	}

// 	req.Method, req.URL, req.Proto, err = parseRequestFirstLine(firstLine)
// 	if err != nil {
// 		return nil, true, err
// 	}

// 	if req.Proto != "HTTP/1.1" || req.URL[0] != '/' || req.Method != "GET" {
// 		return nil, true, fmt.Errorf("1.400")
// 	}

// 	req.Headers = make(map[string]string)

// 	hasHost := false
// 	for {
// 		line, err := ReadLine(br)
// 		key, value, err := parseRequestRestLine(line)
// 		if line == "" {
// 			break
// 		}
// 		if err != nil {
// 			return nil, true, err
// 		}
// 		if key == "Host" { // host exists but it does not have value ==> 200 OK
// 			hasHost = true
// 			req.Host = value
// 		} else if key == "Connection" && value == "close" {
// 			req.Close = true
// 		} else {
// 			req.Headers[key] = value
// 		}
// 	}

// 	// host is missing: request is invalid
// 	if !hasHost {
// 		return nil, true, fmt.Errorf("2.400")
// 	}

// 	return req, true, err
// }

// func parseRequestFirstLine(line string) (string, string, string, error) {
// 	fields := strings.SplitN(line, " ", 3) // split into 2 parts
// 	// check the length of my fields
// 	if len(fields) != 3 {
// 		return "", "", "", fmt.Errorf("Could not parse the request line")
// 	}

// 	return fields[0], fields[1], fields[2], nil
// }

// func parseRequestRestLine(line string) (string, string, error) {
// 	fields := strings.SplitN(line, ":", 2) // split into 2 parts
// 	// check the length of my fields
// 	if len(fields) != 2 {
// 		return "", "", fmt.Errorf("Could not parse the request line")
// 	}

// 	return CanonicalHeaderKey(strings.TrimSpace(fields[0])), strings.TrimSpace(fields[1]), nil
// }

// // ReadLine reads a single line ending with "\r\n" from br,
// // striping the "\r\n" line end from the returned string.
// // If any error occurs, data read before the error is also returned.
// // You might find this function useful in parsing requests.
//
//	func ReadLine(br *bufio.Reader) (string, error) {
//		var line string
//		for {
//			s, err := br.ReadString('\n')
//			line += s
//			// Return the error
//			if err != nil {
//				return line, err
//			}
//			// Return the line when reaching line end
//			if strings.HasSuffix(line, "\r\n") {
//				// Striping the line end
//				line = line[:len(line)-2]
//				return line, nil
//			}
//		}
//	}
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

func MakeRequest(br *bufio.Reader) (req *Request, MoreBytes bool, err error) {
	req = &Request{}
	req.Headers = make(map[string]string)

	// Read start line
	line, err := ReadLine(br)
	if err != nil {
		return nil, false, err
	}

	req.Method, req.URL, req.Proto, err = parseRequestLine(line)
	if err != nil {
		return nil, true, badStringError("malformed start line", line)
	}

	//A well-formed URL always starts with a /character. If the slash is missing, send back a 400 error
	if req.Method != "GET" || string(req.URL[0]) != "/" || req.Proto != "HTTP/1.1" {
		return nil, true, badStringError("invalid method", req.Method) //400 error
	}

	setHost := false
	for {
		line, err := ReadLine(br)
		if err != nil {
			return nil, true, err
		}
		if line == "" {
			// This marks header end
			break
		}
		//Host (required, 400 client error if not present)
		//Connection (optional, if set to “close” then server should close connection with
		//the client after sending response for this request)
		// Any request headers not in the proper form (e.g., missing a colon), should
		//signal a 400 error
		header := strings.SplitN(line, ":", 2)
		if len(header) != 2 {
			return nil, true, badStringError("invalid header", line)
		}
		key := CanonicalHeaderKey(strings.TrimSpace(header[0]))
		value := strings.TrimSpace(header[1])

		if key == "Host" {
			setHost = true
			req.Host = value
		} else if key == "Connection" && value == "close" {
			req.Close = true
		} else {
			req.Headers[key] = value
		}
	}
	if !setHost {
		return nil, true, fmt.Errorf("400")
	}

	return req, true, nil
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its individual parts.
func parseRequestLine(line string) (string, string, string, error) {
	fields := strings.SplitN(line, " ", 3)
	if len(fields) != 3 {
		return "", "", "", fmt.Errorf("could not parse the request line, got fields %v", fields)
	}
	return fields[0], fields[1], fields[2], nil
}

// func validMethod(method string, URL string, Proto string) bool {
// 	return method == "GET" || string(URL[0]) == "/" || Proto == "HTTP/1.1"
// }

func badStringError(what, val string) error {
	return fmt.Errorf("%s %q", what, val)
}
