package cache

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

type CacheObject struct {
	URL             *url.URL
	StatusCode      int
	Body            []byte
	ResponseHeaders http.Header
	RequestHeaders  http.Header
	Method          string
}

const (
	HeaderCacheUnreachable = "UNREACHABLE"
	HeaderCacheHit         = "HIT"
	HeaderCacheMiss        = "MISS"
	HeaderCacheDynamic     = "DYNAMIC"
)

type CacheLevel int

const (
	Disabled CacheLevel = iota
	IgnoreQuery
	Query
	NoQuery
	Standard
	All
)

type RedisAdapter struct {
	store *redisCache.Cache
}

func NewRedisAdapter(opt *redis.Options) *RedisAdapter {
	ropt := redis.Options(*opt)
	r := redis.NewClient(&ropt)
	return &RedisAdapter{redisCache.New(&redisCache.Options{
		Redis:      r,
		LocalCache: redisCache.NewTinyLFU(1000, time.Minute),
	})}
}

func (a *RedisAdapter) Get(key uint64) (obj *CacheObject, exists bool) {
	ctx := context.TODO()

	wanted := &CacheObject{}
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

func (c CacheObject) Cache(a *RedisAdapter, key uint64, level CacheLevel, expiration time.Duration) (err error) {

	if err != nil {
		panic(err)
	}

	err = a.store.Set(&redisCache.Item{
		Key:   KeyAsString(key),
		Value: c,
		TTL:   expiration,
	})
	if err != nil {
		panic(err)
	}
	return nil
}
