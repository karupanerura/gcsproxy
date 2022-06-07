package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/karupanerura/gcsproxy"
)

const defaultPort = "8080"

func main() {
	proxy, err := gcsproxy.CreateHTTPHandler(context.Background(), gcsproxy.Config{
		BucketName:   mustEnv("GCS_PROXY_BUCKET"),
		PathPrefix:   getEnv("GCS_PROXY_PATH_PREFIX", "/"),
		IndexFile:    getEnv("GCS_PROXY_INDEX_FILE", ""),
		NotFoundPath: getEnv("GCS_PROXY_NOT_FOUND_PATH", ""),
		BufByteSize:  0,
	})
	if err != nil {
		log.Printf("failed to create proxy from env: %s", err)
	}

	port := getEnv("PORT", defaultPort)
	svr := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: proxy,
	}
	if err := svr.ListenAndServe(); err == nil || errors.Is(err, http.ErrServerClosed) {
		return // ok
	} else {
		log.Printf("ListenAndServe: %v", err)
	}
}

func getEnv(name, defaultValue string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return defaultValue
}

func mustEnv(name string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	panic(fmt.Sprintf("env: %s is required", name))
}
