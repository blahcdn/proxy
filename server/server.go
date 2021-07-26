package server

import (
	"flag"
	"log"

	"net/http"

	"github.com/blahcdn/proxy/server/cache"
	"github.com/blahcdn/proxy/server/handler"
	"github.com/go-redis/redis/v8"
)

var port string

func StartServer() {
	flag.StringVar(&port, "p", ":5000", "port to bind to")

	store := cache.NewRedisAdapter(&redis.Options{
		Network: "unix",
		Addr:    "/var/run/redis/redis.sock",
	})
	// store := handler.NewAdapter(&redis.Options{
	// 	Addr: "127.0.0.1:6379",
	// })

	flag.Parse()
	handler.AddHost("localhost:5000", false, "192.168.219.102:3001")

	log.Fatal(http.ListenAndServeTLS(port, "127.0.0.1+1.pem", "127.0.0.1+1-key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rc := handler.InitReqCall(w, r)
		rc.ProxyHandler(store)
	})))

}
