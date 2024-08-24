package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Album represents data about a music album.
type Album struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
