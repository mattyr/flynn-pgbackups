package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/jackc/pgx"
	"github.com/flynn/flynn/pkg/postgres"
)

var db *postgres.DB

func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	db, err = setupTestDb()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	os.Exit(m.Run())
}

func TestKeepingBackups(t *testing.T) {
	// daily
	d := time.Now().Add(-3 * 24 * time.Hour)
	b := &Backup{StartedAt: &d}
	if shouldDeleteBackup(b) {
		t.Error("should keep the daily backup")
	}
	// weekly
	d = time.Now().Add(-8 * 24 * time.Hour)
	for d.Weekday() != time.Sunday {
		d = d.Add(24 * time.Hour)
	}
	b = &Backup{StartedAt: &d}
	if shouldDeleteBackup(b) {
		t.Error("should keep the weely backup")
	}
	// monthly
	d, _ = time.Parse(time.RFC822, "01 Jan 15 01:00 UTC")
	b = &Backup{StartedAt: &d}
	if shouldDeleteBackup(b) {
		t.Error("should keep the monthly backup")
	}
	// other
	d, _ = time.Parse(time.RFC822, "02 Jan 14 01:00 UTC")
	b = &Backup{StartedAt: &d}
	if !shouldDeleteBackup(b) {
		t.Error("should delete other backup")
	}
}

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
