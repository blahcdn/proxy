package compress

import (
	"bytes"

	"github.com/andybalholm/brotli"
)

func StaticCompressBr(b []byte) []byte {
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()

}
