package server

import (
	"flag"
	"fmt"
	"time"

	"net/http"

	"github.com/blahcdn/proxy/server/handler"
	"github.com/go-redis/redis/v8"

	"github.com/lucas-clemente/quic-go/http3"
)

var port string

func StartServer() {
	flag.StringVar(&port, "p", ":5000", "an int")

	store := handler.NewAdapter(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	flag.Parse()
	handler.AddHost("localhost:5000", false, "192.168.219.102:3001", 10*time.Minute)
	handler.AddHost("localhost:8000", false, "192.168.219.102:3001", 10*time.Minute)

	//AddHost("localhost:4000", false, "192.168.219.102:3001", 5*time.Minute)

	// Start server
	go http3.ListenAndServeQUIC(port, "127.0.0.1+1.pem", "127.0.0.1+1-key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := handler.InitReqCall(w, r)
		rc.ProxyHandler(store)
	}))

	http.ListenAndServeTLS(port, "127.0.0.1+1.pem", "127.0.0.1+1-key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("alt-svc", fmt.Sprintf(`h3-27=":%[1]v"; ma=86400, h3-28=":%[1]v"; ma=86400, h3-29="%[1]v"; ma=86400, h3=%[1]v"; ma=86400`, port))
		rc := handler.InitReqCall(w, r)
		rc.ProxyHandler(store)
	}))

	//  http.ListenAndServeTLS(":4000", "127.0.0.1+1.pem", "127.0.0.1+1-key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	// 	rc := handler.InitReqCall(w, r)
	// 	rc.ProxyHandler(store)
	// }))
	// http.ListenAndServe(":4000", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	rc := handler.InitReqCall(w, r)
	// 	rc.ProxyHandler(store)
	// }))
	// http.ListenAndServeTLS(":4000", "127.0.0.1+1.pem", "127.0.0.1+1-key.pem", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	rc := handler.InitReqCall(w, r)
	// 	rc.ProxyHandler(store)
	// }))

}

// Handler~, "Hello, World!")
