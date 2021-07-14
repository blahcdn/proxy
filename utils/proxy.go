package proxy

import (
	"net/url"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/valyala/fasthttp"
)

var client = &fasthttp.HostClient{
	NoDefaultUserAgentHeader: true,
	DisablePathNormalizing:   true,
}

func GetOrigin(h string) string {
	if h == "127.0.0.1:3000" {
		return "http://192.168.219.102:3001"
	}
	return ""

}

// Do performs the given http request and fills the given http response.
// This method can be used within a fiber.Handler
func Do(c *fiber.Ctx, h string) (err error) {

	client.Addr = h
	// Set request and response
	req := c.Request()
	res := c.Response()

	// Don't proxy "Connection" header
	req.Header.Del(fiber.HeaderConnection)

	req.SetRequestURI(utils.UnsafeString(req.RequestURI()))

	// Forward request
	if err = client.DoRedirects(req, res, 10); err != nil {
		return err
	}

	// Don't proxy "Connection" header
	res.Header.Del(fiber.HeaderConnection)

	// Return nil to end proxying if no error
	return nil
}
func DoHead(c *fiber.Ctx, h string) (resp *fasthttp.ResponseHeader, err error) {
	client.Addr = h
	// Set request and response
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	// Don't proxy "Connection" header
	req.Header.Del(fiber.HeaderConnection)
	req.Header.SetMethod("HEAD")
	req.SetRequestURI(c.Request().URI().String())

	req.SetRequestURI(utils.UnsafeString(req.RequestURI()))

	// Forward request
	if err = client.Do(req, res); err != nil {
		return nil, err
	}
	// Don't proxy "Connection" header
	res.Header.Del(fiber.HeaderConnection)

	// Return nil to end proxying if no error
	return &res.Header, nil
}

func Middleware() fiber.Handler {
	mux := &sync.RWMutex{}

	//todo: use unix sockets for less latency
	store := NewAdapter(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	return func(c *fiber.Ctx) (err error) {

		key := GenerateKey(c.Request().URI().String())
		if c.Method() != fiber.MethodGet {
			c.Set("x-cache", CacheUnreachable)
		} else {
			e, exists := store.Get(key)
			resp := c.Response()
			if exists {
				resp.SetBodyRaw(e.Body)

				resp.SetStatusCode(e.StatusCode)
				if len(e.ContentEncoding) > 0 {
					c.Response().Header.SetBytesV(fiber.HeaderContentEncoding, e.ContentEncoding)
				}
				resp.Header.SetContentTypeBytes(e.ContentType)
				c.Set("x-cache", CacheHit)
				for k, v := range e.Headers {
					c.Set(k, v)
				}
				return nil

			} else {
				// Continue stack, return err to Fiber if exist
				if err := c.Next(); err != nil {
					return err
				}

				origin := GetOrigin(string(c.Request().Host()))
				if origin == "" {
					c.Response().SetStatusCode(403)
					c.SendString("Direct access prohibited")
					return nil
				}
				u, err := url.Parse(origin)
				if err != nil {
					panic(err)
				}
				if err := Do(c, u.Host); err != nil {
					return err
				}
				// Lock entry and unlock when finished
				mux.Lock()
				defer mux.Unlock()

				store.Set(key, c, All, 10*time.Minute)
				c.Set("x-cache", CacheMiss)

				return nil
			}

		}
		return nil
	}
}
