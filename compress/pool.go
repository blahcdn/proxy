package compress

import (
	"compress/gzip"
	"sync"

	"github.com/andybalholm/brotli"
)

var (
	brp [brotli.BestCompression - brotli.BestSpeed + 2]*sync.Pool
	gzp [gzip.BestCompression - gzip.BestSpeed + 2]*sync.Pool
)

type Encoding int

const (
	Br Encoding = iota
	Gz
)

func init() {
	for i := brotli.BestSpeed; i <= brotli.BestCompression; i++ {
		addLevelPool(i, Br)
	}
	addLevelPool(brotli.DefaultCompression, Br)
	for i := gzip.BestSpeed; i <= gzip.BestCompression; i++ {
		addLevelPool(i, Gz)
	}
	addLevelPool(gzip.DefaultCompression, Gz)

}

func addLevelPool(level int, enc Encoding) {
	switch enc {
	case Br:
		brp[poolIndex(level, Br)] = &sync.Pool{
			New: func() interface{} {
				return brotli.NewWriterLevel(nil, level)
			},
		}
	case Gz:
		gzp[poolIndex(level, Gz)] = &sync.Pool{
			New: func() interface{} {
				// NewWriterLevel only returns error on a bad level, we are guaranteeing
				// that this will be a valid level so it is okay to ignore the returned
				// error.1
				w, _ := gzip.NewWriterLevel(nil, level)

				return w
			},
		}
	}

}
func poolIndex(level int, enc Encoding) int {
	if enc == Br {
		return level
	} else {
		// gzip.DefaultCompression == -1, so we need to treat it special.
		if level == gzip.DefaultCompression {
			return gzip.BestCompression - gzip.BestSpeed + 1
		}
		return level - gzip.BestSpeed
	}

}
