package telemetry

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// OTelMiddleware traces HTTP requests with OpenTelemetry.
func OTelMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, serviceName,
			otelhttp.WithFilter(func(r *http.Request) bool {
				switch r.URL.Path {
				case "/healthz", "/metrics":
					return false
				default:
					return true
				}
			}),
		)
	}
}

// HTTPLogMiddleware emits structured request logs via slog.
func HTTPLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		ctx := r.Context()
		if reqID := middleware.GetReqID(ctx); reqID != "" {
			ctx = WithRequestID(ctx, reqID)
		}

		next.ServeHTTP(ww, r.WithContext(ctx))

		Ctx(ctx).Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.status),
			slog.Int("bytes", ww.bytes),
			slog.String("remote_addr", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// RecoverMiddleware logs panics with slog and returns HTTP 500.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				Ctx(r.Context()).Error("panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
