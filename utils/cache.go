package proxy

import (
	"context"
	"hash/fnv"
	"strconv"
	"time"

	redisCache "github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

type Object struct {
	StatusCode      int
	ContentType     []byte
	Body            []byte
	ContentEncoding []byte
	Host            string
	Headers         map[string]string
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

func (a *Adapter) Set(key uint64, c *fiber.Ctx, level cacheLevel, expiration time.Duration) (err error) {
	req := c.Request()
	res := c.Response()
	// var headers fasthttp.ResponseHeader

	// if level == Disabled {
	// 	return nil
	// }

	// if level == All {
	// 	headers = res.Header
	// } else {
	// 	headers = nil
	// }
	tmpheaders := make(map[string]string)
	res.Header.VisitAll(func(key []byte, value []byte) {
		tmpheaders[string(key)] = string(value)
	})
	obj := &Object{
		ContentType:     res.Header.ContentType(),
		StatusCode:      res.StatusCode(),
		ContentEncoding: c.Response().Header.Peek(fiber.HeaderContentEncoding),
		Body:            res.Body(),
		Host:            string(req.Header.Host()),
		Headers:         tmpheaders,
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
