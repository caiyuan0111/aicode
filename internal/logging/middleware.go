package logging

import (
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// TraceMiddleware injects a trace ID into every request and logs the result.
func TraceMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use incoming trace ID or generate a new one
			traceID := r.Header.Get("X-Trace-Id")
			if traceID == "" {
				traceID = GenerateTraceID()
			}

			// Set response header so the client can retrieve it
			w.Header().Set("X-Trace-Id", traceID)

			// Inject into context
			ctx := WithTraceID(r.Context(), traceID)
			r = r.WithContext(ctx)

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			start := time.Now()
			next.ServeHTTP(rw, r)
			elapsed := time.Since(start)

			// Log the request
			logger.Info("request",
				"trace_id", traceID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", float64(elapsed.Microseconds())/1000.0,
			)
		})
	}
}
