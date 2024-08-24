package postgrestest

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func dsn(addr, user, password, dbName string) string {
	return fmt.Sprintf("postgresql://%s/%s?user=%s&password=%s&sslmode=disable", addr, dbName, user, password)
}

func openDB(addr, user, password, dbName string) (*sql.DB, error) {
	return sql.Open("postgres", dsn(addr, user, password, dbName))
}

func healthCheckDB(addr, user, password, dbName string) (bool, error) {
	db, err := openDB(addr, user, password, dbName)
	if err != nil {
		return false, err
	}
	defer db.Close()
	if err := db.QueryRow("SELECT 1").Err(); err != nil {
		return false, nil
	}
	return true, nil
}
