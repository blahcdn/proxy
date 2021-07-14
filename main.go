package main

import (
	proxy "github.com/blahcdn/proxy/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork: true,
	})

	app.Use(proxy.Middleware())
	app.Use(compress.New())

	app.Listen(":3000")
}
