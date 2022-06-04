package gcsproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
)

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

func CreateProxyFromEnv(ctx context.Context) (http.Handler, error) {

	return createProxy(ctx, &gcsProxyConfig{
		bucketName: mustEnv("GCS_PROXY_BUCKET"),
		pathPrefix: getEnv("GCS_PROXY_PATH_PREFIX", "/"),
		indexFile:  getEnv("GCS_PROXY_INDEX_FILE", ""),
	})
}

type gcsProxyConfig struct {
	bucketName string
	pathPrefix string
	indexFile  string
}

func createProxy(ctx context.Context, config *gcsProxyConfig) (http.Handler, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	return &gcsProxy{
		client: client,
		bucket: client.Bucket(config.bucketName),
		config: config,
	}, nil
}

type gcsProxy struct {
	client *storage.Client
	bucket *storage.BucketHandle
	config *gcsProxyConfig
}

func (p *gcsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, p.config.pathPrefix)
	if key == "" || strings.HasSuffix(key, "/") {
		key += p.config.indexFile
	}

	object := p.bucket.Object(key)
	attrs, err := object.Attrs(r.Context())
	if errors.Is(err, storage.ErrBucketNotExist) {
		http.Error(w, fmt.Sprintf("GCS bucket %q is not found", p.config.bucketName), http.StatusInternalServerError)
		return
	} else if errors.Is(err, storage.ErrObjectNotExist) {
		http.Error(w, fmt.Sprintf("GCS object %q is not found", key), http.StatusInternalServerError)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("object.NewReader for path %q: %v", err), http.StatusBadGateway)
		return
	}

	rdr, err := object.NewReader(r.Context())
	if errors.Is(err, storage.ErrBucketNotExist) {
		http.Error(w, fmt.Sprintf("GCS bucket %q is not found", p.config.bucketName), http.StatusInternalServerError)
		return
	} else if errors.Is(err, storage.ErrObjectNotExist) {
		http.Error(w, fmt.Sprintf("GCS object %q is not found", key), http.StatusInternalServerError)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("object.NewReader for path %q: %v", err), http.StatusBadGateway)
		return
	}

	// set headers
	if v := attrs.Etag; v != "" {
		w.Header().Set("ETag", v)
	}
	if v := attrs.CacheControl; v != "" {
		w.Header().Set("Cache-Control", v)
	}
	if v := attrs.ContentEncoding; v != "" {
		w.Header().Set("Content-Encoding", v)
	}
	if v := attrs.ContentType; v != "" {
		w.Header().Set("Content-Type", v)
	}
	if v := attrs.Size; v != 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(v, 10))
	}
	w.WriteHeader(http.StatusOK)

	const bufSize = 32 * 1024
	var buf [bufSize]byte
	_, err = io.CopyBuffer(w, rdr, buf[:])
	if err == context.Canceled {
		// it means client closed
	} else if err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
