package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// compressible MIME types — only compress text-based and JSON responses.
var compressibleTypes = map[string]bool{
	"application/json":       true,
	"application/javascript": true,
	"text/html":              true,
	"text/css":               true,
	"text/plain":             true,
	"text/xml":               true,
	"application/xml":        true,
	"image/svg+xml":          true,
}

// gzipWriterPool reuses gzip writers to reduce GC pressure under high load.
var gzipWriterPool = sync.Pool{
	New: func() any {
		w, err := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		// usually this will not fail as we are using default compression level but still
		if err != nil {
			panic(err)
		}
		return w
	},
}

// Compress automatically applies Gzip compression to responses when the
// client indicates support via the Accept-Encoding header.
//
// Features:
//   - Respects Accept-Encoding (only compresses if client supports gzip)
//   - Only compresses compressible MIME types (skips images, binaries)
//   - Uses a sync.Pool for gzip writers to minimize allocations
//   - Sets appropriate headers (Content-Encoding, Vary)
//   - Removes Content-Length since compressed size is unknown ahead of time
//
// * If you're behind a CDN or reverse proxy (nginx, Cloudflare) that
// * handles compression, you can skip this middleware and let the proxy do it.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if the client (browser) accepts gzip encoding
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Get a gzip writer from the pool
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)

		defer func() {
			_ = gz.Close()
			gzipWriterPool.Put(gz)
		}()

		cw := &compressWriter{
			ResponseWriter: w,
			gzipWriter:     gz,
		}

		// Set Vary header so caches know the response depends on encoding.
		w.Header().Set("Vary", "Accept-Encoding")

		next.ServeHTTP(cw, r)
	})
}

// compressWriter wraps http.ResponseWriter to transparently compress
// the response body. It delays setting Content-Encoding until Write()
// is called, so it can check the Content-Type first.
type compressWriter struct {
	http.ResponseWriter
	gzipWriter     *gzip.Writer
	headerWritten  bool
	shouldCompress bool
	decided        bool
}

// decide checks if the response should be compressed.
func (cw *compressWriter) decide() {
	if cw.decided {
		return
	}
	cw.decided = true

	contentType := cw.ResponseWriter.Header().Get("Content-Type")
	// Strip parameters like: charset=utf-8
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	if compressibleTypes[contentType] {
		cw.shouldCompress = true
		cw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		// Content-Length is no longer valid since we're compressing and size will change
		cw.ResponseWriter.Header().Del("Content-Length")
	}
}

func (cw *compressWriter) WriteHeader(code int) {
	cw.decide()
	cw.headerWritten = true
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if !cw.headerWritten {
		// If WriteHeader hasn't been called, detect content type from body.
		if cw.ResponseWriter.Header().Get("Content-Type") == "" {
			cw.ResponseWriter.Header().Set("Content-Type", http.DetectContentType(b))
		}
		cw.decide()
	}

	if cw.shouldCompress {
		return cw.gzipWriter.Write(b)
	}

	return cw.ResponseWriter.Write(b)
}

// Flush supports streaming responses. If the underlying writer supports
// http.Flusher, we flush the gzip writer first.
func (cw *compressWriter) Flush() {
	if cw.shouldCompress {
		_ = cw.gzipWriter.Flush()
	}
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (cw *compressWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}
