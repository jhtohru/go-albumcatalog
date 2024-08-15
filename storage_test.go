package albumcatalog

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/assert"
)

func TestPostgresAlbumStorage_Insert(t *testing.T) {
	t.Parallel()

	db, drop := newTmpDB(t)
	defer drop()
	defer db.Close()
	storage := NewPostgresAlbumStorage(db)
	alb := randomAlbum()

	err := storage.Insert(context.Background(), alb)

	assert.Equal(t, alb, findAlbum(t, db, alb.ID))
	assert.Nil(t, err)
}

func TestPostgresAlbumStorage_FindAll(t *testing.T) {
	t.Parallel()

	db, drop := newTmpDB(t)
	defer drop()
	defer db.Close()
	storage := NewPostgresAlbumStorage(db)

	t.Run("no results from empty database", func(t *testing.T) {
		albs, err := storage.FindAll(context.Background(), 0, 100)

		assert.Empty(t, albs)
		assert.ErrorIs(t, err, ErrAlbumNotFound)
	})

	fixture := randomAlbums(5)
	rand.Shuffle(len(fixture), func(i, j int) {
		fixture[i], fixture[j] = fixture[j], fixture[i]
	})
	insertAlbums(t, db, fixture...)

	t.Run("happy path", func(t *testing.T) {
		offset := 1
		limit := 3
		want := fixture[:]
		sort.Slice(want, func(i, j int) bool {
			return strings.ToLower(want[i].Title) < strings.ToLower(want[j].Title)
		})
		want = want[offset : offset+limit]

		albs, err := storage.FindAll(context.Background(), offset, limit)

		assert.Equal(t, albs, want)
		assert.Nil(t, err)
	})

	t.Run("no results from populated database", func(t *testing.T) {
		offset := len(fixture)
		limit := offset + 10

		albs, err := storage.FindAll(context.Background(), offset, limit)

		assert.Empty(t, albs)
		assert.ErrorIs(t, err, ErrAlbumNotFound)
	})
}

func TestPostgresAlbumStorage_FindOne(t *testing.T) {
	t.Parallel()

	db, drop := newTmpDB(t)
	defer drop()
	defer db.Close()

	storage := NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		alb, err := storage.FindOne(context.Background(), uuid.New())

		assert.Empty(t, alb)
		assert.ErrorIs(t, err, ErrAlbumNotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		want := randomAlbum()
		insertAlbums(t, db, want)

		alb, err := storage.FindOne(context.Background(), want.ID)

		assert.Equal(t, want, alb)
		assert.Nil(t, err)
	})
}

func TestPostgresAlbumStorage_Update(t *testing.T) {
	t.Parallel()

	db, drop := newTmpDB(t)
	defer drop()
	defer db.Close()
	storage := NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		err := storage.Update(context.Background(), randomAlbum())

		assert.ErrorIs(t, err, ErrAlbumNotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		albOutdated := randomAlbum()
		insertAlbums(t, db, albOutdated)
		albUpdated := randomAlbum()
		albUpdated.ID = albOutdated.ID

		err := storage.Update(context.Background(), albUpdated)

		assert.Nil(t, err)
		assert.Equal(t, albUpdated, findAlbum(t, db, albUpdated.ID))
	})
}

func TestPostgresAlbumStorage_Remove(t *testing.T) {
	t.Parallel()

	db, drop := newTmpDB(t)
	defer drop()
	defer db.Close()
	storage := NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		err := storage.Remove(context.Background(), uuid.New())

		assert.ErrorIs(t, err, ErrAlbumNotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		alb := randomAlbum()
		insertAlbums(t, db, alb)

		err := storage.Remove(context.Background(), alb.ID)

		assert.Nil(t, err)
		assert.False(t, albumExists(t, db, alb.ID))
	})
}

func newTmpDB(t *testing.T) (db *sql.DB, drop func() error) {
	t.Helper()

	psqlAddr := postgresAddress(t)
	dbName := "tmpdb_" + strings.ToLower(randomString(30))
	if err := createDB(psqlAddr, dbName); err != nil {
		t.Fatalf("Failed to create the database: %v", err)
	}
	db, err := openDB(psqlAddr, dbName)
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}
	if err := migrateDB(db); err != nil {
		t.Fatalf("Failed to migrate the database: %v", err)
	}
	drop = func() error {
		return dropDB(psqlAddr, dbName)
	}
	return db, drop
}

func postgresAddress(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("POSTGRES_ADDR")
	if addr == "" {
		t.Fatalf("POSTGRES_ADDR env var is not set")
	}
	return addr
}

func openDB(dbmsAddr, dbName string) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgresql://postgres@%s/%s?sslmode=disable", dbmsAddr, dbName)

	return sql.Open("postgres", dsn)
}

func createDB(dbmsADdr, dbName string) error {
	db, err := openDB(dbmsADdr, "postgres")
	if err != nil {
		return err
	}
	defer db.Close()
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err = db.Exec(query)

	return err
}

func dropDB(dbmsAddr, dbName string) error {
	db, err := openDB(dbmsAddr, "postgres")
	if err != nil {
		return err
	}
	defer db.Close()
	query := fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbName)
	_, err = db.Exec(query)

	return err
}

func migrateDB(db *sql.DB) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("could not get file path")
	}
	migrationsDir := path.Join(filepath.Dir(file), "migrations/")
	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	return nil
}

func findAlbum(t *testing.T, db *sql.DB, albID uuid.UUID) Album {
	t.Helper()

	query := "SELECT id, title, artist, price, created_at, updated_at FROM album WHERE id = $1"
	row := db.QueryRow(query, albID)
	var alb Album
	err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price, &alb.CreatedAt, &alb.UpdatedAt)
	if err != nil {
		t.Fatalf("Could not find album: %v", err)
	}
	alb.CreatedAt = alb.CreatedAt.Local()
	alb.UpdatedAt = alb.UpdatedAt.Local()

	return alb
}

func insertAlbums(t *testing.T, db *sql.DB, albs ...Album) {
	t.Helper()

	query := "INSERT INTO album (id, title, artist, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
	stmt, err := db.Prepare(query)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	for _, alb := range albs {
		_, err := stmt.Query(alb.ID, alb.Title, alb.Artist, alb.Price, alb.CreatedAt.UTC(), alb.UpdatedAt.UTC())
		if err != nil {
			t.Fatal(err)
		}
	}
}

func albumExists(t *testing.T, db *sql.DB, albID uuid.UUID) bool {
	t.Helper()

	query := "SELECT EXISTS (SELECT 1 FROM album WHERE id = $1)"
	row := db.QueryRow(query, albID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		t.Fatal(err)
	}

	return exists
}
