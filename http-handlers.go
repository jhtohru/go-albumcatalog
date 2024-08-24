package catalog

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Validator can validate itself.
type Validator interface {
	// Valid returns the Validator problems.
	// If Validator is valid, len(problems) == 0.
	Valid() (problems map[string]string)
}

// Validate returns the Validator problems.
// If Validator is valid, len(problems) == 0.
func Validate(v Validator) map[string]string {
	return v.Valid()
}

type request struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Price  int    `json:"price"`
}

// Valid makes request implement Validator.
func (req request) Valid() map[string]string {
	problems := make(map[string]string)
	if req.Title == "" {
		problems["title"] = "is empty"
	}
	if req.Artist == "" {
		problems["artist"] = "is empty"
	}
	if req.Price <= 0 {
		problems["price"] = "is not greater than zero"
	}
	return problems
}

// createAlbumHandler returns an http.Handler to requests to create an album.
func createAlbumHandler(
	albumStorage AlbumStorage,
	logger *slog.Logger,
	validate func(Validator) map[string]string,
	newID func() uuid.UUID,
	timeNow func() time.Time,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract album data from the request.
		req, err := decode[request](r)
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "malformed request body")
			return
		}
		if problems := validate(req); len(problems) > 0 {
			encodeProblems(w, http.StatusBadRequest, "invalid request body", problems)
			return
		}
		// Create a new album and insert into the storage.
		now := timeNow()
		alb := Album{
			ID:        newID(),
			Title:     req.Title,
			Artist:    req.Artist,
			Price:     req.Price,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err = albumStorage.Insert(r.Context(), alb); err != nil {
			logger.Error("inserting album into the storage", "error", err)
			encodeMessage(w, http.StatusInternalServerError, "internal error")
			return
		}
		// Respond with the new album.
		encode(w, http.StatusCreated, alb)
	})
}

// maxAlbumsPageSize is the maximum quantity of albums an album page can have.
const maxAlbumsPageSize = 50

// listAlbumsHandler returns an http.Handler to requests to list albums.
func listAlbumsHandler(albumStorage AlbumStorage, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract page size and page number from the request.
		q := r.URL.Query()
		if !q.Has("page_size") {
			encodeMessage(w, http.StatusBadRequest, "query parameter page_size is missing")
			return
		}
		pageSize, err := strconv.Atoi(q.Get("page_size"))
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "page size is not a valid number")
			return
		}
		if !q.Has("page_number") {
			encodeMessage(w, http.StatusBadRequest, "query parameter page_number is missing")
			return
		}
		pageNumber, err := strconv.Atoi(q.Get("page_number"))
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "page number is not a valid number")
			return
		}
		// Validate page size and page number.
		if pageSize < 1 {
			encodeMessage(w, http.StatusBadRequest, "page size is less than 1")
			return
		}
		if pageSize > maxAlbumsPageSize {
			msg := fmt.Sprintf("page size is greater than %d", maxAlbumsPageSize)
			encodeMessage(w, http.StatusBadRequest, msg)
			return
		}
		if pageNumber < 1 {
			encodeMessage(w, http.StatusBadRequest, "page number is less than 1")
			return
		}
		// Find albums in the storage.
		offset, limit := pageSize*(pageNumber-1), pageSize
		albs, err := albumStorage.FindAll(r.Context(), offset, limit)
		if err != nil {
			switch {
			case errors.Is(err, ErrAlbumNotFound):
				// If no album is found, respond with an empty list and OK status code.
				encode(w, http.StatusOK, []Album{})
			default:
				logger.Error("finding albums in the storage", "error", err)
				encodeMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		// Respond with the found albums.
		encode(w, http.StatusOK, albs)
	})
}

// getAlbumHandler returns an http.Handler to requests to get an album.
func getAlbumHandler(albumStorage AlbumStorage, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract album id from the request.
		albID, err := uuid.Parse(r.PathValue("album_id"))
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "malformed album id")
			return
		}
		// Find album in the storage.
		alb, err := albumStorage.FindOne(r.Context(), albID)
		if errors.Is(err, ErrAlbumNotFound) {
			encodeMessage(w, http.StatusNotFound, "album not found")
			return
		}
		if err != nil {
			logger.Error("finding one album in the storage", "error", err)
			encodeMessage(w, http.StatusInternalServerError, "internal error")
			return
		}
		// Respond with the found album.
		encode(w, http.StatusOK, alb)
	})
}

// updateAlbumHandler returns an http.Handler to requests to update an album.
func updateAlbumHandler(
	albumStorage AlbumStorage,
	logger *slog.Logger,
	validate func(Validator) map[string]string,
	timeNow func() time.Time,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract album id from the request.
		albID, err := uuid.Parse(r.PathValue("album_id"))
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "malformed album id")
			return
		}
		// Extract updated album data from request.
		req, err := decode[request](r)
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "malformed request body")
			return
		}
		if problems := validate(req); len(problems) > 0 {
			encodeProblems(w, http.StatusBadRequest, "invalid request body", problems)
			return
		}
		// Find album in the storage.
		alb, err := albumStorage.FindOne(r.Context(), albID)
		if err != nil {
			switch {
			case errors.Is(err, ErrAlbumNotFound):
				encodeMessage(w, http.StatusNotFound, "album not found")
			default:
				logger.Error("finding one album in the storage", "error", err)
				encodeMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		// Update album in the storage.
		alb.Title = req.Title
		alb.Artist = req.Artist
		alb.Price = req.Price
		alb.UpdatedAt = timeNow()
		if err := albumStorage.Update(r.Context(), alb); err != nil {
			switch {
			case errors.Is(err, ErrAlbumNotFound):
				encodeMessage(w, http.StatusNotFound, "album not found")
			default:
				logger.Error("updating album in the storage", "error", err)
				encodeMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		// Respond with the updated album.
		encode(w, http.StatusOK, alb)
	})
}

// deleteAlbumHandler returns an http.Handler to requests to delete an album.
func deleteAlbumHandler(albumStorage AlbumStorage, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract album id from the request.
		albID, err := uuid.Parse(r.PathValue("album_id"))
		if err != nil {
			encodeMessage(w, http.StatusBadRequest, "malformed album id")
			return
		}
		// Find album in the storage.
		alb, err := albumStorage.FindOne(r.Context(), albID)
		if err != nil {
			switch {
			case errors.Is(err, ErrAlbumNotFound):
				encodeMessage(w, http.StatusNotFound, "album not found")
			default:
				logger.Error("finding one album in the storage", "error", err)
				encodeMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		// Remove album from the storage.
		if err := albumStorage.Remove(r.Context(), albID); err != nil {
			switch {
			case errors.Is(err, ErrAlbumNotFound):
				encodeMessage(w, http.StatusNotFound, "album not found")
			default:
				logger.Error("removing album from the storage", "error", err)
				encodeMessage(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		// Respond with the removed album.
		encode(w, http.StatusOK, alb)
	})
}
