package handler

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/blahcdn/proxy/compress"
	"github.com/blahcdn/proxy/headers"
	"github.com/blahcdn/proxy/response"
	"github.com/blahcdn/proxy/server/cache"
)

type RequestCall struct {
	Response *response.ResponseWriter
	Request  *http.Request
}

func (rc RequestCall) serveFromCache(e *cache.CacheObject) {

	res := rc.Response
	res.CopyHeaders(e.ResponseHeaders)
	res.Header().Set(cacheHeader, cache.HeaderCacheHit)
	res.WriteHeader(e.StatusCode)
	res.Write(e.Body)
}

func ConvertRequestCallToCacheObj(rc RequestCall) (*cache.CacheObject, uint64) {
	var obj *cache.CacheObject
	var key uint64
	encoding := compress.NegotiateContentEncoding(rc.Request, []string{"br", "gzip"})
	clonedReqHeaders := rc.Request.Header.Clone()
	clonedResHeaders := rc.Response.Header().Clone()

	if len(rc.Response.Content) > 7000 && !strings.HasSuffix(rc.Request.URL.Path, ".woff2") {

		switch encoding {

		case "br":
			rc.Response.Header().Set(headers.ContentEncoding, "br")
			rc.Response.Compressed = true
			rc.Response.Header().Del(headers.ContentLength)
			obj = &cache.CacheObject{
				ResponseHeaders: clonedResHeaders,
				RequestHeaders:  clonedReqHeaders,
				Body:            compress.StaticCompressBr(rc.Response.Content),
				URL:             rc.Request.URL,
				Method:          rc.Request.Method,
				StatusCode:      rc.Response.StatusCode,
			}
			key = rc.GenerateKey("br")
			return obj, key

		}

	}
	key = rc.GenerateKey("plain")
	obj = &cache.CacheObject{
		ResponseHeaders: clonedResHeaders,
		RequestHeaders:  clonedReqHeaders,
		Body:            rc.Response.Content,
		URL:             rc.Request.URL,
		Method:          rc.Request.Method,
		StatusCode:      rc.Response.StatusCode,
	}

	return obj, key

}

func (rc RequestCall) GenerateKey(enc string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(enc + rc.Request.URL.String()))

	return hash.Sum64()
}

func InitReqCall(res http.ResponseWriter, req *http.Request) RequestCall {
	return RequestCall{
		Response: response.NewResponseWriter(res),
		Request:  req,
	}
}

func patchProxyTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		Dial: func(network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, 15*time.Second)
			if err != nil {
				return conn, err
			}

			return conn, err
		},
		DisableKeepAlives: false,
	}
}
