package main

import (
	"flag"
	"os"
	"testing"
	"time"

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
