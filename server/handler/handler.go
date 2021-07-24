package handler

func (rc RequestCall) serveFromCache(e *Object) {

	res := rc.Response

	res.CopyHeaders(e.Headers)
	contentEncoding := e.Headers.Get("Content-Encoding")
	if len(contentEncoding) > 0 {
		res.Header().Set("Content-Encoding", contentEncoding)
	}
	res.Header().Set("Content-Type", e.Headers.Get("Content-Type"))
	res.Header().Set("asdf", "123")
	res.Header().Set(cacheHeader, CacheHit)
	res.Write(e.Body)

}
