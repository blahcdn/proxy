package proxy

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Config defines the config for middleware.
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool

	// Expiration is the time that an cached response will live
	//
	// Optional. Default: 1 * time.Minute
	Expiration time.Duration

	// CacheHeader header on response header, indicate cache status, with the following possible return value
	//
	// hit, miss, unreachable
	//
	// Optional. Default: X-Cache
	CacheHeader string

	// CacheControl enables client side caching if set to true
	//
	// Optional. Default: false
	CacheControl bool

	// Key allows you to generate custom keys, by default c.Path() is used
	//
	// Default: func(c *fiber.Ctx) string {
	//   return c.Path()
	// }
	KeyGenerator func(*fiber.Ctx) string

	// Store is used to store the state of the middleware
	//
	// Default: an in memory store for this process only
	Storage fiber.Storage

	// Deprecated, use Storage instead
	Store fiber.Storage

	// Deprecated, use KeyGenerator instead
	Key func(*fiber.Ctx) string
}

// ConfigDefault is the default config
var ConfigDefault = Config{
	Next:         nil,
	Expiration:   1 * time.Minute,
	CacheHeader:  "X-Cache",
	CacheControl: false,
	KeyGenerator: func(c *fiber.Ctx) string {
		return c.Path()
	},
	Storage: nil,
}
