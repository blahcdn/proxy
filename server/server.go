package server

import (
	"flag"
	"log"

	"net/http"

	"github.com/blahcdn/proxy/compress"
	"github.com/blahcdn/proxy/server/cache"
	"github.com/blahcdn/proxy/server/handler"
	"github.com/go-redis/redis/v8"
)

var port string

func StartServer() {
	flag.StringVar(&port, "p", ":4000", "port to bind to")

	store := cache.NewRedisAdapter(&redis.Options{
		Network: "unix",
		Addr:    "/var/run/redis/redis.sock",
	})
	// store := handler.NewAdapter(&redis.Options{
	// 	Addr: "127.0.0.1:6379",
	// })

	flag.Parse()

	handler.AddHost("localhost:1337", false, "192.168.219.102:3001")

	handler.AddHost("127.0.0.1:4000", false, "192.168.219.102:3001")
	handler.AddHost("127.0.0.1:1337", false, "192.168.219.102:3000")
	handler.AddHost("localhost:4000", false, "192.168.219.102:3001")

	log.Fatal(http.ListenAndServe(port, compress.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rc := handler.InitReqCall(w, r)
		rc.ProxyHandler(store)
	}))))
}
