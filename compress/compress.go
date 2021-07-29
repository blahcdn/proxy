package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/blahcdn/proxy/headers"
	"github.com/blahcdn/proxy/response"
)

const (
	BrEncoding      = "br"
	GzipEncoding    = "gzip"
	DeflateEncoding = "deflate"
)

func (cwr *CompressedResponseWriter) Write(b []byte) (int, error) {
	if cwr.Writer != nil {
		cwr.writeHeader()
		return cwr.Writer.Write(b)
	}
	println(cwr.ResponseWriter.Compressed)
	if cwr.ResponseWriter.Compressed {
		cwr.writeHeader()
		return cwr.ResponseWriter.Write(b)
	}

	// If we have already decided not to use GZIP, immediately passthrough.
	if cwr.ignore {
		cwr.writeHeader()
		return cwr.ResponseWriter.Write(b)
	}

	cwr.buf = append(cwr.buf, b...)
	// Don't compress short pieces of data since it won't improve performance. Ignore compressed data too
	if len(b) < 1400 {

		return len(b), cwr.startPlain()
	}
	switch cwr.encoding {
	case Br:
		cwr.Header().Set(headers.ContentEncoding, "br")
	case Gz:

		cwr.Header().Set(headers.ContentEncoding, "gzip")
	}
	h := cwr.ResponseWriter.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", http.DetectContentType(b))
	}
	h.Del("Content-Length")
	defer cwr.Writer.Close()
	return cwr.Writer.Write(b)
}

type CompressedResponseWriter struct {
	Writer io.WriteCloser
	*response.ResponseWriter
	http.Hijacker
	http.Flusher
	buf      []byte
	brIndex  int // index for BrWriterPools
	gzIndex  int
	ignore   bool // If true, then we immediately passthru writes to the underlying ResponseWriter.
	code     int
	encoding Encoding
}

// Reset - Reset the stored content of ResponseWriter.
func (cwr *CompressedResponseWriter) Reset() {
	cwr.code = 0
	cwr.buf = make([]byte, 0)
}

func (cwr *CompressedResponseWriter) WriteHeader(c int) {
	cwr.ResponseWriter.Header().Del("Content-Length")
	if cwr.code == 0 {
		cwr.code = c
	}
}
func (cwr *CompressedResponseWriter) writeHeader() {
	if cwr.code != 0 {
		cwr.ResponseWriter.WriteHeader(cwr.code)
	}

}

func (cwr *CompressedResponseWriter) Header() http.Header {
	return cwr.ResponseWriter.Header()
}

type flusher interface {
	Flush() error
}

func (w *CompressedResponseWriter) Flush() {
	// Flush compressed data if compressor supports it.
	f, ok := w.Writer.(flusher)
	if !ok {
		return
	}
	f.Flush()
	// Flush HTTP response.
	if w.Flusher != nil {
		w.Flusher.Flush()
	}
}

func NewCompressedResponseWriter(w http.ResponseWriter, r *http.Request) *CompressedResponseWriter {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		hijacker = nil
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		flusher = nil
	}
	gzIndex := poolIndex(gzip.DefaultCompression, Gz)
	brIndex := poolIndex(brotli.DefaultCompression, Br)
	wr := &CompressedResponseWriter{ResponseWriter: response.NewResponseWriter(w), Hijacker: hijacker, brIndex: brIndex, gzIndex: gzIndex, Flusher: flusher}

	wr.Writer = wr.init(r)

	wr.Reset()
	return wr
}

func (cwr *CompressedResponseWriter) init(r *http.Request) io.WriteCloser {

	// if cwr.Header().Get("Content-Type") == "" {

	// 	return nopCloser{cwr.ResponseWriter.ResponseWriter}
	// }

	if cwr.Header().Get("Vary") == "" {
		cwr.Header().Set(headers.Vary, headers.AcceptEncoding)
	}
	encoding := NegotiateContentEncoding(r, []string{"br", "gzip"})
	if cwr.code != 0 {
		cwr.writeHeader()
		// Ensure that no other WriteHeader's happen
		cwr.code = 0
	}
	switch encoding {
	case "br":
		cwr.encoding = Br
		brw := brp[cwr.brIndex].Get().(*brotli.Writer)
		brw.Reset(cwr.ResponseWriter)
		return brw

	case "gzip":
		cwr.encoding = Gz
		cwr.Header().Set(headers.ContentEncoding, "gzip")

		gzw := gzp[cwr.gzIndex].Get().(*gzip.Writer)

		gzw.Reset(cwr.ResponseWriter)
		return gzw
	}

	return nopCloser{cwr.ResponseWriter}
}

// Close will close the gzip.Writer and will put it back in the gzipWriterPool.
func (cwr *CompressedResponseWriter) Close() error {

	if cwr.ignore {
		return nil
	}

	if cwr.Writer == nil {
		// compression not triggered yet, write out regular response.
		err := cwr.startPlain()
		// Returns the error if any at write.
		if err != nil {
			err = fmt.Errorf("gziphandler: write to regular responseWriter at close gets error: %q", err.Error())
		}
		return err
	}

	err := cwr.Writer.Close()
	switch cwr.encoding {
	case Br:
		brp[cwr.brIndex].Put(cwr.Writer)
	case Gz:
		gzp[cwr.gzIndex].Put(cwr.Writer)
	}

	cwr.Writer = nil
	return err
}

func (w *CompressedResponseWriter) startPlain() error {
	w.writeHeader()
	w.ignore = true
	// If Write was never called then don't call Write on the underlying ResponseWriter.
	if w.buf == nil {
		return nil
	}
	n, err := w.ResponseWriter.Write(w.buf)
	w.buf = nil
	// This should never happen (per io.Writer docs), but if the write didn't
	// accept the entire buffer but returned no specific error, we have no clue
	// what's going on, so abort just to be safe.
	if err == nil && n < len(w.buf) {
		err = io.ErrShortWrite
	}
	return err
}

func newCompressHandler(level int) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headers.Vary, headers.AcceptEncoding)
			encoding := NegotiateContentEncoding(r, []string{"br", "gzip"})
			switch encoding {
			case "br":
				cwr := NewCompressedResponseWriter(w, r)
				defer cwr.Close()
				h.ServeHTTP(cwr, r)
			}
			h.ServeHTTP(w, r)

		})
	}
}

func CompressHandler(h http.Handler) http.Handler {
	wrapper := newCompressHandler(brotli.DefaultCompression)
	return wrapper(h)
}
