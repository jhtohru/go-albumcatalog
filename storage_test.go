package catalog_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	catalog "github.com/jhtohru/go-album-catalog"
	"github.com/jhtohru/go-album-catalog/internal/postgrestest"
	"github.com/jhtohru/go-album-catalog/internal/random"
	"github.com/jhtohru/go-album-catalog/internal/runutil"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	exitStatus, err := run(ctx, m)
	cancel()
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(exitStatus)
}

var postgresTest *postgrestest.Postgres

func run(ctx context.Context, m *testing.M) (int, error) {
	var (
		postgresAddr      = os.Getenv("POSTGRES_ADDR")
		postgresUser      = runutil.GetenvDefault("POSTGRES_USER", "postgres")
		postgresPassword  = runutil.GetenvDefault("POSTGRES_PASSWORD", "password")
		postgresDefaultDB = runutil.GetenvDefault("POSTGRES_DEFAULT_DB", "postgres")
	)
	if postgresAddr == "" {
		addr, terminate, err := postgrestest.SpinUpContainer(ctx, postgresUser, postgresPassword, postgresDefaultDB)
		if err != nil {
			return 1, fmt.Errorf("spinning up postgres container: %w", err)
		}
		defer terminate(ctx)
		postgresAddr = addr
	}
	var err error
	postgresTest, err = postgrestest.New(ctx, postgresAddr, postgresUser, postgresPassword, postgresDefaultDB)
	if err != nil {
		return 1, fmt.Errorf("starting postgres test: %w", err)
	}
	defer postgresTest.Terminate()
	return m.Run(), nil
}

func TestPostgresAlbumStorage_Insert(t *testing.T) {
	t.Parallel()

	db := postgresTest.CreateDBOrFailNow(t)
	defer db.Close()
	storage := catalog.NewPostgresAlbumStorage(db)
	alb := randomAlbum()

	err := storage.Insert(context.Background(), alb)

	assert.Equal(t, alb, findAlbum(t, db, alb.ID))
	assert.Nil(t, err)
}

func TestPostgresAlbumStorage_FindAll(t *testing.T) {
	t.Parallel()

	db := postgresTest.CreateDBOrFailNow(t)
	defer db.Close()
	storage := catalog.NewPostgresAlbumStorage(db)

	t.Run("no results from empty database", func(t *testing.T) {
		albs, err := storage.FindAll(context.Background(), 0, 100)

		assert.Empty(t, albs)
		assert.ErrorIs(t, err, catalog.ErrAlbumNotFound)
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
		assert.ErrorIs(t, err, catalog.ErrAlbumNotFound)
	})
}

func TestPostgresAlbumStorage_FindOne(t *testing.T) {
	t.Parallel()

	db := postgresTest.CreateDBOrFailNow(t)
	defer db.Close()
	storage := catalog.NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		alb, err := storage.FindOne(context.Background(), uuid.New())

		assert.Empty(t, alb)
		assert.ErrorIs(t, err, catalog.ErrAlbumNotFound)
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

	db := postgresTest.CreateDBOrFailNow(t)
	defer db.Close()
	storage := catalog.NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		err := storage.Update(context.Background(), randomAlbum())

		assert.ErrorIs(t, err, catalog.ErrAlbumNotFound)
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

	db := postgresTest.CreateDBOrFailNow(t)
	defer db.Close()
	storage := catalog.NewPostgresAlbumStorage(db)

	t.Run("album not found", func(t *testing.T) {
		err := storage.Remove(context.Background(), uuid.New())

		assert.ErrorIs(t, err, catalog.ErrAlbumNotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		alb := randomAlbum()
		insertAlbums(t, db, alb)

		err := storage.Remove(context.Background(), alb.ID)

		assert.Nil(t, err)
		assert.False(t, albumExists(t, db, alb.ID))
	})
}

// randomAlbum returns a randomly generated Album.
func randomAlbum() catalog.Album {
	return catalog.Album{
		ID:        uuid.New(),
		Title:     random.String(20 + rand.IntN(20)),
		Artist:    random.String(20 + rand.IntN(20)),
		Price:     rand.IntN(100000),
		CreatedAt: random.Time(),
		UpdatedAt: random.Time(),
	}
}

// randomAlbums returns a slice containing n randomly generated Albums.
func randomAlbums(n int) []catalog.Album {
	albs := make([]catalog.Album, n)
	for i := range albs {
		albs[i] = randomAlbum()
	}
	return albs
}

func findAlbum(t *testing.T, db *sql.DB, albID uuid.UUID) catalog.Album {
	t.Helper()

	query := "SELECT id, title, artist, price, created_at, updated_at FROM album WHERE id = $1"
	row := db.QueryRow(query, albID)
	var alb catalog.Album
	err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price, &alb.CreatedAt, &alb.UpdatedAt)
	if err != nil {
		t.Fatalf("Could not find album: %v", err)
	}
	alb.CreatedAt = alb.CreatedAt.Local()
	alb.UpdatedAt = alb.UpdatedAt.Local()

	return alb
}

func insertAlbums(t *testing.T, db *sql.DB, albs ...catalog.Album) {
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
