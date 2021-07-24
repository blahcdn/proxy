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
	StatusCode int
	Content    []byte
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	lwr := &ResponseWriter{ResponseWriter: w}
	lwr.Reset()
	return lwr
}

// Reset - Reset the stored content of ResponseWriter.
func (wr *ResponseWriter) Reset() {
	wr.StatusCode = 0
	wr.Content = make([]byte, 0)
}

// Write - ResponseWriter's Write method decorator.
func (wr *ResponseWriter) Write(p []byte) (int, error) {
	wr.Content = append(wr.Content, p...)

	return wr.ResponseWriter.Write(p)
}

func (wr *ResponseWriter) WriteBody(page string) bool {
	pageByte := []byte(page)
	sent, err := wr.ResponseWriter.Write(pageByte)

	return sent > 0 && err == nil
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
