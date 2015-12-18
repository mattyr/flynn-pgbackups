package main

import (
	"fmt"
	"io"
	"os"

	"github.com/flynn/flynn/pkg/postgres"
	"github.com/jackc/pgx"
)

func setupTestDb() (*postgres.DB, error) {
	dbname := "pgbackupstest"

	if err := setupDatabase(dbname); err != nil {
		return nil, err
	}

	return getDatabase(dbname)
}

func getDatabase(dbname string) (*postgres.DB, error) {
	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     os.Getenv("PGHOST"),
			Database: dbname,
			User:     os.Getenv("PGUSER"),
			Password: os.Getenv("PGPASSWORD"),
		},
	})
	if err != nil {
		return nil, err
	}
	db := postgres.New(pgxpool, nil)

	return db, nil
}

func setupDatabase(dbname string) error {
	if os.Getenv("PGDATABASE") != "" {
		dbname = os.Getenv("PGDATABASE")
	} else {
		os.Setenv("PGDATABASE", dbname)
	}
	if os.Getenv("PGSSLMODE") == "" {
		os.Setenv("PGSSLMODE", "disable")
	}
	if os.Getenv("PGHOST") == "" {
		os.Setenv("PGHOST", "localhost")
	}

	connConfig := pgx.ConnConfig{
		Host:     os.Getenv("PGHOST"),
		Database: "postgres",
		User:     os.Getenv("PGUSER"),
		Password: os.Getenv("PGPASSWORD"),
	}

	db, err := pgx.Connect(connConfig)
	if err != nil {
		return err
	}

	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbname)); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname)); err != nil {
		return err
	}
	return nil
}

type dummyStore struct {
}

func newDummyStore() *dummyStore {
	return &dummyStore{}
}

func (*dummyStore) DownloadUrl(string, string) (string, error) {
	return "", nil
}

func (*dummyStore) Put(appId string, backupId string, r io.Reader) error {
	return nil
}

func (*dummyStore) Delete(appId string, backupId string) error {
	return nil
}
