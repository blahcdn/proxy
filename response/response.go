package response

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

type ResponseWriter struct {
	http.ResponseWriter
	http.Hijacker
	Content    []byte
	StatusCode int
	Compressed bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	wr := &ResponseWriter{ResponseWriter: w}
	wr.Reset()
	return wr
}

// WriteHeader - ResponseWriter's WriteHeader method decorator.
func (wr *ResponseWriter) WriteHeader(statusCode int) {
	wr.StatusCode = statusCode
	//wr.ResponseWriter.WriteHeader(statusCode)
}

// Reset - Reset the stored content of ResponseWriter.
func (wr *ResponseWriter) Reset() {
	wr.Content = make([]byte, 0)
	wr.StatusCode = 0
}

// Write - ResponseWriter's Write method decorator.
func (wr *ResponseWriter) Write(b []byte) (int, error) {
	wr.Content = append(wr.Content, b...)
	return wr.ResponseWriter.Write(b)
}

func (wr *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := wr.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}

	return hj.Hijack()
}

func (wr *ResponseWriter) CopyHeaders(src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			wr.Header().Set(k, v)
		}
	}
}
