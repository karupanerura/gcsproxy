package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/karupanerura/gcsproxy"
	"gocloud.dev/server"
)

const defaultPort = "8080"

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = defaultPort
	}

	proxy, err := gcsproxy.CreateProxyFromEnv(context.Background())
	if err != nil {
		log.Printf("failed to create proxy from env: %s", err)
	}

	svr := server.New(proxy, &server.Options{})
	svr.ListenAndServe(net.JoinHostPort("0.0.0.0", port))
}
