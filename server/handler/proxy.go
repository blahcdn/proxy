package handler

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/blahcdn/proxy/response"
	"github.com/labstack/echo/v4"
	//...
)

const cacheHeader = "x-cache-status"

type SSLOption int

const (
	OFF SSLOption = iota
	LAX
	FULL
)

type RequestCall struct {
	Response *response.ResponseWriter
	Request  *http.Request
}

type Target struct {
	URL       *url.URL
	handler   *httputil.ReverseProxy
	expiresOn time.Time
}

var hostProxy = struct {
	sync.RWMutex
	targets map[string]*Target
}{targets: make(map[string]*Target)}

func AddHost(host string, isWebsocket bool, target string, ttl time.Duration) (err error) {
	var remoteUrl *url.URL
	if isWebsocket {
		remoteUrl, err = url.Parse("wss://" + target)
	} else {
		remoteUrl, err = url.Parse("http://" + target)
	}

	if err != nil {
		log.Println("target parse fail:", err)
		return
	}

	// Lock map hostProxy since maps are not safe for concurrent use
	hostProxy.Lock()
	defer hostProxy.Unlock()

	targetQuery := remoteUrl.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = remoteUrl.Scheme
		req.URL.Host = remoteUrl.Host

		req.URL.Path, req.URL.RawPath = joinURLPath(remoteUrl, req.URL)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	proxy := &httputil.ReverseProxy{Director: director}
	hostProxy.targets[host] = &Target{handler: proxy, expiresOn: time.Now().Add(ttl)}

	return nil
}

func checkExpired(host string) (err error) {
	if target, ok := hostProxy.targets[host]; ok {
		hostProxy.Lock()

		defer hostProxy.Unlock()

		// Amount of time passed after the handler expired >=0
		if time.Since(target.expiresOn) >= 0 {
			println("expired")

		}

	}
	return nil
}

func (rc *RequestCall) ProxyHandler(store *Adapter) {
	r := rc.Request
	w := rc.Response

	host := r.Host

	key := GenerateKey(rc.Request.URL.String())
	if target, ok := hostProxy.targets[host]; ok {

		// Fix header
		if r.Header.Get(echo.HeaderXRealIP) == "" {
			r.Header.Set(echo.HeaderXRealIP, r.RemoteAddr)
		}
		if r.Header.Get(echo.HeaderXForwardedProto) == "" {
			r.Header.Set(echo.HeaderXForwardedProto, r.URL.Scheme)
		}
		if isWebsocket(r) && r.Header.Get(echo.HeaderXForwardedFor) == "" { // For HTTP, it is automatically set by Go HTTP reverse proxy.
			r.Header.Set(echo.HeaderXForwardedFor, r.RemoteAddr)
		}

		// Proxy
		switch {
		case isWebsocket(r):
			proxyRaw(target, w, r).ServeHTTP(w, r)
		case r.Header.Get(echo.HeaderAccept) == "text/event-stream":
		default:
			host := r.Host
			if r.Method != "GET" {
				w.Header().Set(cacheHeader, CacheMiss)
			} else {

				e, exists := store.Get(key)
				if exists {
					rc.serveFromCache(e)
					return

				}
			}

			fn, ok := hostProxy.targets[host]
			if ok {
				fn.handler.ServeHTTP(w, r)
				go rc.Cache(store, key, cacheLevel(FULL), 5*time.Minute)
				go checkExpired(host)
				return
			} else {
				http.Error(w, "Direct access prohibited", http.StatusForbidden)

			}
		}
		return
	}
	http.Error(w, "403: Host forbidden "+host, http.StatusForbidden)

}

// Note: Websockets should not be cached
func proxyRaw(t *Target, inw http.ResponseWriter, inr *http.Request) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := inw.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		in, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy raw, hijack error (url: %s) %v:", t.URL, err), http.StatusInternalServerError)
			return
		}
		defer in.Close()

		out, err := net.Dial("tcp", t.URL.Host)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy raw, dial error (url: %s) %v:", t.URL, err), http.StatusBadGateway)

			return
		}
		defer out.Close()

		// Write header
		err = r.Write(out)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy raw, request header copy error=%v, url=%s", t.URL, err), http.StatusBadGateway)
			return
		}

		errCh := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err = io.Copy(dst, src)
			errCh <- err
		}

		go cp(out, in)
		go cp(in, out)
		err = <-errCh
		if err != nil && err != io.EOF {
			print(fmt.Errorf("proxy raw, copy body error=%v, url=%s", t.URL, err))
		}
	})
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func isWebsocket(r *http.Request) bool {
	upgrade := r.Header.Get("Upgrade")
	return strings.EqualFold(upgrade, "websocket")

}

func InitReqCall(res http.ResponseWriter, req *http.Request) RequestCall {
	return RequestCall{
		Response: response.NewResponseWriter(res),
		Request:  req,
	}
}

func (rc *RequestCall) Cache(store *Adapter, key uint64, level cacheLevel, ttl time.Duration) {
	store.Set(key, rc, level, ttl)
}
