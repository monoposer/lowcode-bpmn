package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"lowcode-bpmn/internal/api"
	"lowcode-bpmn/internal/engine"
	pgstore "lowcode-bpmn/internal/store/postgres"
)

func main() {
	addr := getEnv("HTTP_ADDR", ":8080")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set (e.g. postgres://user:pass@host:5432/dbname)")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open postgres connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	rootCtx := context.Background()
	store, err := pgstore.NewStore(rootCtx, db)
	if err != nil {
		log.Fatalf("failed to init postgres store: %v", err)
	}

	eng := engine.NewEngine(store, nil)
	if getEnvBool("ASYNC_EXECUTION", false) {
		eng.SetAsync(true)
		log.Println("async execution enabled")
	}

	workerCtx, workerCancel := context.WithCancel(rootCtx)
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
		log.Printf("lowcode-bpmn listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	workerCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
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
