package gcsproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

type Config struct {
	BucketName   string
	PathPrefix   string
	IndexFile    string
	NotFoundPath string
	BufByteSize  int
}

type Option interface {
	apply(*gcsProxy) error
}

type optionFunc func(*gcsProxy) error

func (f optionFunc) apply(p *gcsProxy) error {
	return f(p)
}

func WithClient(client *storage.Client) Option {
	return optionFunc(func(p *gcsProxy) error {
		p.client = client
		return nil
	})
}

const defaultBufSize = 32 * 1024 // 32Kbyte

func CreateHTTPHandler(ctx context.Context, config Config, opts ...Option) (http.Handler, error) {
	if config.BufByteSize == 0 {
		config.BufByteSize = defaultBufSize
	}

	proxy := &gcsProxy{
		config: config,
		bufPool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, config.BufByteSize)
				return &b
			},
		},
	}
	for _, opt := range opts {
		if err := opt.apply(proxy); err != nil {
			return nil, fmt.Errorf("Option.apply: %w", err)
		}
	}

	// connect default client
	if proxy.client == nil {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage.NewClient: %w", err)
		}
		proxy.client = client
	}

	// init bucket
	proxy.bucket = proxy.client.Bucket(config.BucketName)

	return proxy, nil
}

type gcsProxy struct {
	config Config

	bufPool *sync.Pool
	client  *storage.Client
	bucket  *storage.BucketHandle
}

func (p *gcsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// accept
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	condition, err := parseCacheCondition(r.Header)
	if err != nil {
		log.Printf("invalid cache condition: %v", err)
		http.Error(w, "invalid conditional request", http.StatusBadRequest)
		return
	}

	bodyRange, err := parseRange(r.Header.Get("Range"))
	if err != nil {
		log.Printf("invalid range %q: %v", r.Header.Get("Range"), err)
		http.Error(w, "invalid range", http.StatusBadRequest)
		return
	}

	req := proxyRequest{
		path:           r.URL.Path,
		cacheCondition: condition,
		bodyRange:      bodyRange,
		allowGziped:    isGzipAllowed(r.Header.Get("Accept-Encoding")),
		onlyHead:       r.Method == http.MethodHead,
		statusCode:     http.StatusOK,
	}

	p.proxy(r.Context(), w, &req)
}

func isGzipAllowed(ae string) bool {
	for ae != "" {
		i := strings.IndexByte(ae, ',')
		if i == -1 {
			i = len(ae)
		}

		e := textproto.TrimString(ae[:i])
		if strings.EqualFold(e, "gzip") || strings.EqualFold(e, "x-gzip") {
			return true
		}

		ae = ae[i:]
		if len(ae) > 0 {
			ae = ae[1:]
		}
	}

	return false
}

type proxyRequest struct {
	path           string
	cacheCondition cacheCondition
	bodyRange      bodyRange
	allowGziped    bool
	onlyHead       bool
	statusCode     int
}

func (p *gcsProxy) proxy(ctx context.Context, w http.ResponseWriter, req *proxyRequest) {
	key := strings.TrimPrefix(req.path, p.config.PathPrefix)
	if key == "" || strings.HasSuffix(key, "/") {
		key += p.config.IndexFile
	}

	object := p.bucket.Object(key).ReadCompressed(req.allowGziped)
	attrs, err := object.Attrs(ctx)
	if errors.Is(err, storage.ErrBucketNotExist) {
		http.Error(w, fmt.Sprintf("GCS bucket %q is not found", p.config.BucketName), http.StatusInternalServerError)
		return
	} else if errors.Is(err, storage.ErrObjectNotExist) || (attrs != nil && !attrs.Deleted.IsZero()) {
		if p.config.NotFoundPath != "" && p.config.NotFoundPath != req.path {
			req := proxyRequest(*req)
			req.path = p.config.NotFoundPath
			req.statusCode = http.StatusNotFound
			p.proxy(ctx, w, &req)
			return
		}

		http.Error(w, fmt.Sprintf("GCS object %q is not found (path: %q)", key, req.path), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("object.Attrs for path %q: %v", req.path, err), http.StatusBadGateway)
		return
	}

	// set headers
	setHeadersByAttrs(w.Header(), req, attrs)
	if req.cacheCondition.ETag != nil && attrs.Etag != "" && !req.cacheCondition.ETag.Match(attrs.Etag, !req.bodyRange.isAll()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if req.cacheCondition.Time != nil && !attrs.Updated.IsZero() && !req.cacheCondition.Time.Match(attrs.Updated) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if req.onlyHead {
		w.WriteHeader(req.statusCode)
		return
	}

	rdr, err := object.NewRangeReader(ctx, req.bodyRange.offset, req.bodyRange.length)
	if err != nil {
		http.Error(w, fmt.Sprintf("object.NewReader for path %q: %v", req.path, err), http.StatusBadGateway)
		return
	}
	defer rdr.Close()
	setHeadersByReader(w.Header(), req, rdr) // overwrite headers

	// write header
	statusCode := req.statusCode
	if !req.bodyRange.isAll() && statusCode == http.StatusOK {
		statusCode = http.StatusPartialContent
	}
	w.WriteHeader(statusCode)

	bufP := p.bufPool.Get().(*[]byte)
	defer p.bufPool.Put(bufP)

	_, err = io.CopyBuffer(w, rdr, *bufP)
	if errors.Is(err, context.Canceled) {
		// it means client closed
	} else if err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func setHeadersByAttrs(h http.Header, req *proxyRequest, attrs *storage.ObjectAttrs) {
	if v := attrs.Etag; v != "" {
		if req.bodyRange.isAll() {
			h.Set("ETag", strconv.Quote(v))
		} else {
			h.Set("ETag", "W/"+strconv.Quote(v))
		}
	}
	if v := attrs.Updated; !v.IsZero() {
		h.Set("Last-Modified", formatIMFfixdate(v))
	}
	if v := attrs.CacheControl; v != "" {
		h.Set("Cache-Control", v)
	}
	if v := attrs.ContentType; v != "" {
		h.Set("Content-Type", v)
	}
	if v := attrs.Size; v != 0 {
		h.Set("Content-Length", strconv.FormatInt(v, 10))
	}
}

func setHeadersByReader(h http.Header, req *proxyRequest, rdr *storage.Reader) {
	if v := rdr.Attrs.CacheControl; v != "" {
		h.Set("Cache-Control", v)
	}
	if v := rdr.Attrs.ContentEncoding; v != "" {
		h.Set("Content-Encoding", v)
	}
	if v := rdr.Attrs.ContentType; v != "" {
		h.Set("Content-Type", v)
	}
	if v := rdr.Attrs.LastModified; !v.IsZero() {
		h.Set("Last-Modified", formatIMFfixdate(v))
	}

	if req.bodyRange.isAll() {
		h.Set("Content-Length", strconv.FormatInt(rdr.Attrs.Size, 10))
	} else {
		h.Set("Content-Length", strconv.FormatInt(rdr.Remain(), 10))
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rdr.Attrs.StartOffset, rdr.Attrs.StartOffset+rdr.Remain(), rdr.Attrs.Size))
	}
}

func formatIMFfixdate(t time.Time) string {
	if t.Location() != time.UTC {
		t = t.UTC()
	}

	s := t.Format(time.RFC1123)
	if strings.HasSuffix(s, "UTC") {
		return s[:len(s)-3] + "GMT"
	}

	return s
}
