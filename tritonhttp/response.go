package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
)

type Response struct {
	Proto      string // e.g. "HTTP/1.1"
	StatusCode int    // e.g. 200
	StatusText string // e.g. "OK"

	// Headers stores all headers to write to the response.
	Headers map[string]string
	// date, last-modified, content type, content length, connection

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	// Hint: you might need this to handle the "Connection: Close" requirement
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// I've already got the data type of my response, but it's a structure, with fields like proto and statuscode
// How do I read it back to my connection
func (res *Response) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)

	if err := res.WriteStatusLine(bw); err != nil {
		return err
	}

	if err := res.WriteHeaders(bw); err != nil {
		return err
	}

	if err := res.WriteBody(bw); err != nil {
		return err
	}

	// important: to make sure all of your content is right back into your connection, if not, you may receive partial response
	if err := bw.Flush(); err != nil {
		return nil
	}

	return nil
}

func (res *Response) WriteStatusLine(bw *bufio.Writer) error {

	if res.StatusCode == 200 {
		res.StatusText = "OK"
	} else if res.StatusCode == 400 {
		res.StatusText = "Bad Request"
	} else if res.StatusCode == 404 {
		res.StatusText = "Not Found"
	}

	statusLine := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, res.StatusText)

	if _, err := bw.WriteString(statusLine); err != nil {
		return err
	}

	// if err := bw.Flush(); err != nil {
	// 	return nil
	// }
	return nil
}

func (res *Response) WriteHeaders(bw *bufio.Writer) error {
	sortedKeys := make([]string, 0, len(res.Headers))

	for key := range res.Headers {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		headerLine := fmt.Sprintf("%v: %v\r\n", key, res.Headers[key])

		if _, err := bw.WriteString(headerLine); err != nil {
			return err
		}
	}

	if _, err := bw.WriteString("\r\n"); err != nil { // between headers and body
		return err
	}
	// if err := bw.Flush(); err != nil {
	// 	return nil
	// }
	return nil
}

func (res *Response) WriteBody(bw *bufio.Writer) error {
	var content []byte
	var err error
	if res.FilePath != "" {
		if content, err = os.ReadFile(res.FilePath); err != nil {
			return err
		}
		if _, err := bw.Write(content); err != nil {
			return nil
		}
	}

	// if err := bw.Flush(); err != nil {
	// 	return nil
	// }
	return nil
}
