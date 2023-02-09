package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// VirtualHosts contains a mapping from host name to the docRoot path
	// (i.e. the path to the directory to serve static files from) for
	// all virtual hosts that this server supports
	VirtualHosts map[string]string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	// Hint: Validate all docRoots
	if err := s.ValidateServerSetup(); err != nil { // check the directory before everything
		return err
	}

	// Hint: create your listen socket and spawn off goroutines per incoming client
	// server should now start to listen on the configured address
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	// making sure the listener is closed when we exit
	defer ln.Close()

	// accept cnnections forever, because I should keep receiving requests from clients and keep responding responses
	// keep going, so I use a endless for loop to accept new connections
	for {
		conn, err := ln.Accept() // through listen socket accept
		if err != nil {
			return err
		}
		// it must be go routine, because I want my server to handle each connection in parallel,
		// so that no connection will block the future connections
		go s.HandleConnection(conn)
	}
}

func (s *Server) ValidateServerSetup() error {
	// we need to make sure the dirctory exits
	for _, value := range s.VirtualHosts {
		fi, err := os.Stat(value)
		if err != nil { // the directory does not exist
			return err
		}

		if !fi.IsDir() { // it is just single file
			return err
		}
	}
	// valid, return nil
	return nil
}

func (s *Server) HandleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)

	for {
		// set out a timeout
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Printf("Failed to set timeout for connection %v", conn)
			_ = conn.Close()
			return
		}

		// read next request from the client
		req, contentReceived, err := MakeRequest(br)

		// error 1: client has closed the conn ==> io.EOF (还没有timeout就已经读完了，valid)
		if errors.Is(err, io.EOF) {
			log.Printf("Connection closed by %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}

		// error 2: timeout from the server ==> net.Error
		if err0, ok := err.(net.Error); ok && err0.Timeout() {
			if !contentReceived { // read nothing
				log.Printf("Connection To %v timed out", conn.RemoteAddr()) // RemoteAddr returns the remote network address
				_ = conn.Close()
				return
			}
			// read partial
			res := &Response{}
			res.HandleBadRequest()
			_ = res.Write(conn) // 之后都关了 write的error检不检查无所谓
			_ = conn.Close()
			return
		}

		// error 3: malformed/invalid request
		// Handle te request which is not a GET and immediatel  close the connection and return
		if err != nil {
			res := &Response{}
			res.HandleBadRequest()
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// handle good request
		log.Printf("Handle Good Request")
		res := s.HandleGoodRequest(req)
		err = res.Write(conn)
		if err != nil {
			return
		}

		// check if close
		if req.Close {
			_ = conn.Close()
			return
		}
	}
}

func (res *Response) init(req *Request) {
	res.Proto = "HTTP/1.1"
	res.Request = req
	res.Headers = make(map[string]string)
	res.Headers["Date"] = FormatTime(time.Now())
	if req != nil {
		if req.URL[len(req.URL)-1] == '/' {
			req.URL = req.URL + "index.html" // if index.html/ + index.html ==> 404
		}
		if req.Close {
			res.Headers["Connection"] = "close"
		}
	}
}

func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	res = &Response{}
	res.init(req)

	absPath := filepath.Join(s.VirtualHosts[req.Host], req.URL) // path only exist when request is good
	absPath = filepath.Clean(absPath)

	// abs（加了url肯定长一些）里面有没有包含docroot
	if strings.Contains(absPath, s.VirtualHosts[req.Host]) { // 200 -> HandleOK
		res.HandleOK(req, absPath)
	} else { // 404 -> HandleNotHound
		res.HandleNotFound(req)
	}
	return res
}

func (res *Response) HandleBadRequest() {
	// 400
	res.init(nil)
	res.StatusCode = 400
	res.Request = nil
	res.FilePath = "" // set the filepath to be new
}

func (res *Response) HandleNotFound(req *Request) {
	// 404
	res.StatusCode = 404
	res.FilePath = "" // set the filepath to be new
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	res.StatusCode = 200
	res.FilePath = path

	stats, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		res.HandleNotFound(req)
		return
	}
	res.Headers["Last-Modified"] = FormatTime(stats.ModTime())
	res.Headers["Content-Type"] = MIMETypeByExtension(filepath.Ext(path))
	res.Headers["Content-Length"] = strconv.FormatInt(stats.Size(), 10)
	fmt.Println(res.Headers["Content-Length"])
}

// HandleBadRequest prepares res to be a 405 Method Not allowed response
