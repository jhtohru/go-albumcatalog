package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/pressly/goose/v3"
)

type Postgres struct {
	addr, user, password string
	defaultDB            *sql.DB
}

var ErrTerminated = fmt.Errorf("postgres is terminated")

func New(ctx context.Context, addr, user, password, defaultDBName string) (*Postgres, error) {
	defaultDB, err := openDB(addr, user, password, defaultDBName)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		addr:      addr,
		user:      user,
		password:  password,
		defaultDB: defaultDB,
	}, nil
}

func (p *Postgres) CreateDBOrFailNow(t *testing.T) *sql.DB {
	t.Helper()
	if p.defaultDB == nil {
		t.Fatalf("postgres is terminated")
	}
	db, err := p.CreateDB()
	if err != nil {
		t.Fatalf("Creating a database: %v\n", err)
	}
	return db
}

func (p *Postgres) CreateDB() (*sql.DB, error) {
	if p.defaultDB == nil {
		return nil, ErrTerminated
	}
	randomDBName := "tmpdb_" + randomString(30)
	return p.createDB(randomDBName)
}

func (p *Postgres) createDB(dbName string) (*sql.DB, error) {
	query := fmt.Sprintf("CREATE DATABASE %q", dbName)
	if _, err := p.defaultDB.Exec(query); err != nil {
		return nil, err
	}
	db, err := openDB(p.addr, p.user, p.password, dbName)
	if err != nil {
		return nil, err
	}
	// Migrate database.
	if err := goose.Up(db, "migrations"); err != nil {
		p.dropDB(dbName)
		db.Close()
		return nil, err
	}
	return db, nil
}

func (p *Postgres) DropDB(db *sql.DB) error {
	if p.defaultDB == nil {
		return ErrTerminated
	}
	row := p.defaultDB.QueryRow("SELECT current_database()")
	var dbName string
	if err := row.Scan(&dbName); err != nil {
		return err
	}
	return p.dropDB(dbName)
}

func (p *Postgres) dropDB(dbName string) error {
	_, err := p.defaultDB.Exec("DROP DATABASE $1 WITH (FORCE)", dbName)
	return err
}

func (p *Postgres) Terminate() error {
	if p.defaultDB == nil {
		return ErrTerminated
	}
	err := p.defaultDB.Close()
	p.defaultDB = nil
	return err
}

func randomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = letters[rand.IntN(len(letters))]
	}

	return string(b)
}
