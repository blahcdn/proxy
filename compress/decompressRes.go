package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

const GZIPEncoding string = "gzip"

var pool = gzipDecompressPool()

func gzipDecompressPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			// create with an empty reader (but with GZIP header)
			w, err := gzip.NewWriterLevel(ioutil.Discard, gzip.BestSpeed)
			if err != nil {
				return err
			}

			b := new(bytes.Buffer)
			w.Reset(b)
			w.Flush()
			w.Close()

			r, err := gzip.NewReader(bytes.NewReader(b.Bytes()))
			if err != nil {
				return err
			}
			return r
		},
	}
}

func DecompressRes(res *http.Response) (err error) {
	switch res.Header.Get(echo.HeaderContentEncoding) {
	case GZIPEncoding:
		b := res.Body

		i := pool.Get()
		gr, ok := i.(*gzip.Reader)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, i.(error).Error())
		}

		if err := gr.Reset(b); err != nil {
			pool.Put(gr)
			if err == io.EOF { //ignore if body is empty
				return nil
			}
			return err
		}
		var buf bytes.Buffer
		io.Copy(&buf, gr)

		gr.Close()
		pool.Put(gr)

		b.Close() // http.Request.Body is closed by the Server, but because we are replacing it, it must be closed here

		r := ioutil.NopCloser(&buf)
		res.Header.Del(echo.HeaderContentEncoding)
		res.Body = r
	}
	return nil
}
