package main

import (
	"flag"
	"os"
	"testing"

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
