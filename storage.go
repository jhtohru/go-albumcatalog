package catalog

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// AlbumStorage representes an album storage.
type AlbumStorage interface {
	// Insert inserts an Album into the storage.
	Insert(ctx context.Context, alb Album) error
	// FindAll finds all Albums into the storage within offset and limit. It
	// returns ErrAlbumNotFound if no Album was found in the storage within
	// offset and limit.
	FindAll(ctx context.Context, offset, limit int) ([]Album, error)
	// FindOne finds a single Album in the storage. It returns ErrAlbumNotFound
	// if there is no Album in the storage whose ID is equal to id.
	FindOne(ctx context.Context, id uuid.UUID) (Album, error)
	// Update updates the single Album in the storage whose ID is equal to
	// alb.ID setting its state equal to the alb state. It returns
	// ErrAlbumNotFound if there is no Album in the storage whose ID is equal to
	// id.
	Update(ctx context.Context, alb Album) error
	// Remove removes the single Album in the storage whose ID is equal to id.
	// It returns ErrAlbumNotFound if there is no Album in the storage whose ID
	// is equal to id.
	Remove(ctx context.Context, id uuid.UUID) error
}

// ErrAlbumNotFound is returned when the required album was not found in the
// AlbumStorage.
var ErrAlbumNotFound = errors.New("album not found")

type pgAlbumStorage struct {
	db *sql.DB
}

// NewPostgresAlbumStorage returns a new AlbumStorage that uses Postgres to
// manage data
func NewPostgresAlbumStorage(db *sql.DB) AlbumStorage {
	return &pgAlbumStorage{
		db: db,
	}
}

func (s *pgAlbumStorage) Insert(ctx context.Context, alb Album) error {
	query := `
		INSERT INTO
			album (id, title, artist, price, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6)`
	_, err := s.db.QueryContext(ctx, query,
		alb.ID,
		alb.Title,
		alb.Artist,
		alb.Price,
		alb.CreatedAt.UTC(),
		alb.UpdatedAt.UTC(),
	)

	return err
}

func (s *pgAlbumStorage) FindAll(ctx context.Context, offset, limit int) ([]Album, error) {
	query := `
		SELECT
			id, title, artist, price, created_at, updated_at
		FROM
			album
		ORDER BY
			title ASC
		OFFSET
			$1
		LIMIT
			$2`
	rows, err := s.db.QueryContext(ctx, query, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albs []Album
	for rows.Next() {
		alb, err := scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		albs = append(albs, alb)
	}
	if len(albs) == 0 {
		return nil, ErrAlbumNotFound
	}

	return albs, nil
}

func (s *pgAlbumStorage) FindOne(ctx context.Context, id uuid.UUID) (Album, error) {
	query := `
		SELECT
			id, title, artist, price, created_at, updated_at
		FROM
			album
		WHERE
			id = $1`
	row := s.db.QueryRowContext(ctx, query, id)
	alb, err := scanAlbum(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Album{}, ErrAlbumNotFound
	case err != nil:
		return Album{}, err
	}

	return alb, nil
}

func (s *pgAlbumStorage) Update(ctx context.Context, alb Album) error {
	query := `
		UPDATE
			album
		SET
			title = $1,
			artist = $2,
			price = $3,
			created_at = $4,
			updated_at = $5
		WHERE
			id = $6`
	result, err := s.db.ExecContext(ctx, query,
		alb.Title,
		alb.Artist,
		alb.Price,
		alb.CreatedAt.UTC(),
		alb.UpdatedAt.UTC(),
		alb.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrAlbumNotFound
	}

	return nil
}

func (s *pgAlbumStorage) Remove(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM
			album
		WHERE
			id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrAlbumNotFound
	}

	return nil
}

// scanner abstracts *sql.Row and *sql.Rows.
type scanner interface {
	// Scan decode dest from scanner inner data.
	Scan(dest ...any) error
}

// scanAlbum extracts an Album from a scanner.
func scanAlbum(scn scanner) (Album, error) {
	var alb Album
	err := scn.Scan(
		&alb.ID,
		&alb.Title,
		&alb.Artist,
		&alb.Price,
		&alb.CreatedAt,
		&alb.UpdatedAt,
	)
	if err != nil {
		return Album{}, err
	}
	alb.CreatedAt = alb.CreatedAt.Local()
	alb.UpdatedAt = alb.UpdatedAt.Local()
	return alb, nil
}
