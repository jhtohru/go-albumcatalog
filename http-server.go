package catalog

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// NewServer returns a new HTTP server that handles requests to CRUD albums.
func NewServer(
	albumStorage AlbumStorage,
	logger *slog.Logger,
	validate func(Validator) map[string]string,
	newID func() uuid.UUID,
	timeNow func() time.Time,
) http.Handler {
	mux := http.NewServeMux()

	registerRoutes(mux, albumStorage, logger, validate, newID, timeNow)

	return mux
}

// registerRoutes registers HTTP handlers to API routes.
func registerRoutes(
	mux *http.ServeMux,
	albumStorage AlbumStorage,
	logger *slog.Logger,
	validate func(Validator) map[string]string,
	newID func() uuid.UUID,
	timeNow func() time.Time,
) {
	mux.Handle("POST /albums", createAlbumHandler(albumStorage, logger, validate, newID, timeNow))
	mux.Handle("GET /albums", listAlbumsHandler(albumStorage, logger))
	mux.Handle("GET /albums/{album_id}", getAlbumHandler(albumStorage, logger))
	mux.Handle("PUT /albums/{album_id}", updateAlbumHandler(albumStorage, logger, validate, timeNow))
	mux.Handle("DELETE /albums/{album_id}", deleteAlbumHandler(albumStorage, logger))
}
