package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"lowcode-automation/internal/api"
	"lowcode-automation/internal/engine"
	pgstore "lowcode-automation/internal/store/postgres"
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

	e := engine.NewEngine(store)

	// inject engine into request context through middleware
	router := api.NewHTTPRouter()
	router = withEngineMiddleware(router, e)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
	}
}

// withEngineMiddleware attaches the engine to each request context.
func withEngineMiddleware(h http.Handler, e *engine.Engine) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := api.WithEngine(r.Context(), e)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
