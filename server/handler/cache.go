package handler

import (
	"context"
	"hash/fnv"
	"net/http"
	"net/url"
	"strconv"
	"time"

	redisCache "github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
)

type Object struct {
	URL        *url.URL
	StatusCode int
	Body       []byte
	Headers    http.Header
}

const (
	CacheUnreachable = "UNREACHABLE"
	CacheHit         = "HIT"
	CacheMiss        = "MISS"
	CacheDynamic     = "DYNAMIC"
)

type cacheLevel int

const (
	Disabled cacheLevel = iota
	IgnoreQuery
	Query
	NoQuery
	Standard
	All
)

type Adapter struct {
	store *redisCache.Cache
}

func NewAdapter(opt *redis.Options) *Adapter {
	ropt := redis.Options(*opt)
	r := redis.NewClient(&ropt)
	return &Adapter{redisCache.New(&redisCache.Options{
		Redis:      r,
		LocalCache: redisCache.NewTinyLFU(1000, time.Minute),
	})}
}

func (a *Adapter) Set(key uint64, rc *RequestCall, level cacheLevel, expiration time.Duration) (err error) {

	// var headers fasthttp.ResponseHeader

	// if level == Disabled {
	// 	return nil
	// }

	// if level == All {
	// 	headers = res.Header
	// } else {
	// 	headers = nil
	// }
	// tmpheaders := make(map[string]string)
	// res.Header().VisitAll(func(key []byte, value []byte) {
	// 	tmpheaders[string(key)] = string(value)
	// })

	if err != nil {
		panic(err)
	}

	obj := &Object{
		StatusCode: rc.Response.StatusCode,
		Body:       rc.Response.Content,
		URL:        rc.Request.URL,
		Headers:    rc.Response.Header(),
	}
	err = a.store.Set(&redisCache.Item{
		Key:   KeyAsString(key),
		Value: obj,
		TTL:   expiration,
	})
	if err != nil {
		panic(err)
	}
	return nil
}

func (a *Adapter) Get(key uint64) (obj *Object, exists bool) {
	ctx := context.TODO()

	wanted := &Object{}
	if err := a.store.Get(ctx, KeyAsString(key), wanted); err == nil {
		return wanted, true
	}
	return
}

func GenerateKey(URL string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(URL))

	return hash.Sum64()
}

func KeyAsString(key uint64) string {
	return strconv.FormatUint(key, 36)
}
