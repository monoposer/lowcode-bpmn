package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lowcode_bpmn_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lowcode_bpmn_http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	processStartsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "lowcode_bpmn_process_starts_total",
			Help: "Total process instances started",
		},
	)
	taskCompletionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "lowcode_bpmn_task_completions_total",
			Help: "Total user tasks completed",
		},
	)
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		status := strconv.Itoa(ww.Status())
		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

func incProcessStarts() { processStartsTotal.Inc() }
func incTaskCompletions() { taskCompletionsTotal.Inc() }
