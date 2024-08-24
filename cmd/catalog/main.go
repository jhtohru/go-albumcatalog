package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"

	catalog "github.com/jhtohru/go-album-catalog"
	"github.com/jhtohru/go-album-catalog/internal/runutil"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var (
		host          = os.Getenv("SERVER_HOST")
		port          = runutil.GetenvDefault("SERVER_PORT", "8080")
		dsn           = runutil.MustGetenv("DSN")
		willMigrateDB = runutil.GetenvBool("MIGRATE_DB")
	)
	if dsn == "" {
		return fmt.Errorf("postgres dsn is not set")
	}
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	if willMigrateDB {
		if err := goose.Up(db, "migrations"); err != nil {
			return fmt.Errorf("migrating database: %w", err)
		}
	}
	albumStorage := catalog.NewPostgresAlbumStorage(db)
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	logger := slog.New(logHandler)
	srv := catalog.NewServer(
		albumStorage,
		logger,
		catalog.Validate,
		uuid.New,
		time.Now,
	)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: srv,
	}
	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error listening and serving: %v\n", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down the http server: %v\n", err)
		}
	}()
	wg.Wait()

	return nil
}
