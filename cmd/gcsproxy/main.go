package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/karupanerura/gcsproxy"
	"github.com/tg123/go-htpasswd"
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

	handler := proxy
	if f := parseBasicAuth(getEnv("GCS_PROXY_BASIC_AUTH", "")); f != nil {
		next := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || !f.Match(username, password) {
				w.Header().Add("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	port := getEnv("PORT", defaultPort)
	svr := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: handler,
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

func parseBasicAuth(s string) *htpasswd.File {
	if s == "" {
		return nil
	}

	f, err := htpasswd.NewFromReader(strings.NewReader(s), htpasswd.DefaultSystems, func(err error) {
		log.Printf("Invalid GCS_PROXY_BASIC_AUTH: %v", err)
	})
	if err != nil {
		log.Printf("Invalid GCS_PROXY_BASIC_AUTH: %v", err)
		return nil
	}
	return f
}
