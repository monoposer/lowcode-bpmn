package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/api"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/store"
	"github.com/monoposer/lowcode-bpmn/internal/telemetry"
)

func main() {
	ctx := context.Background()
	telCfg := telemetry.LoadConfig()

	shutdownTelemetry, err := telemetry.Init(ctx, telCfg)
	if err != nil {
		slog.Error("failed to init telemetry", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shCtx); err != nil {
			slog.Error("telemetry shutdown failed", slog.String("error", err.Error()))
		}
	}()

	addr := getEnv("HTTP_ADDR", ":8080")

	storeCfg, err := store.LoadConfig()
	if err != nil {
		slog.Error("invalid store config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	st, err := store.Open(ctx, storeCfg)
	if err != nil {
		slog.Error("failed to open store", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := store.Ping(ctx, st); err != nil {
		slog.Error("failed to ping store", slog.String("error", err.Error()))
		os.Exit(1)
	}

	eng := engine.NewEngine(st, nil)
	if getEnvBool("ASYNC_EXECUTION", false) {
		eng.SetAsync(true)
		slog.Info("async execution enabled")
	}

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	worker := engine.NewWorker(eng, getEnvDuration("WORKER_INTERVAL", 500*time.Millisecond))
	go worker.Run(workerCtx)

	deps := api.RouterDeps{Engine: eng}
	router := api.NewHTTPRouter(deps)
	router = withEngineMiddleware(router, deps)
	handler := withCORS(router)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		slog.Info("server listening",
			slog.String("addr", addr),
			slog.String("store", store.Describe(storeCfg)),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	workerCancel()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("shutting down http server")
	if err := srv.Shutdown(shCtx); err != nil {
		slog.Error("server shutdown error", slog.String("error", err.Error()))
	}
}

func withEngineMiddleware(h http.Handler, deps api.RouterDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := api.WithEngine(r.Context(), deps.Engine)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-Id, X-Tenant-ID, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
