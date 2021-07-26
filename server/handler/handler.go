package handler

import "github.com/blahcdn/proxy/server/cache"

func (rc RequestCall) serveFromCache(e *cache.CacheObject) {
	res := rc.Response

	res.CopyHeaders(e.ResponseHeaders)
	res.Header().Set(cacheHeader, cache.HeaderCacheHit)
	res.Write(e.Body)

}
func ConvertRequestCallToCacheObj(rc RequestCall) *cache.CacheObject {
	return &cache.CacheObject{
		ResponseHeaders: rc.Response.Header(),
		RequestHeaders:  rc.Request.Header,
		Body:            rc.Response.Content,
		URL:             rc.Request.URL,
		Method:          rc.Request.Method,
		StatusCode:      rc.Response.StatusCode,
	}
}
