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

	"github.com/jhtohru/go-albumcatalog"
)

func main() {
	fmt.Println(os.Getenv("DSN"))
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	svrHost := os.Getenv("SERVER_HOST")
	svrPort := os.Getenv("SERVER_PORT")
	if svrPort == "" {
		svrPort = "8080"
	}
	dsn := os.Getenv("DSN")
	if dsn == "" {
		return fmt.Errorf("DSN env var not set")
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	albumStorage := albumcatalog.NewPostgresAlbumStorage(db)
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	logger := slog.New(logHandler)
	srv := albumcatalog.NewServer(
		albumStorage,
		logger,
		albumcatalog.Validate,
		uuid.New,
		time.Now,
	)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(svrHost, svrPort),
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
