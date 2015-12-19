package main

import (
	"testing"
	"time"

	"github.com/flynn/flynn/pkg/random"
)

func TestRepo(t *testing.T) {
	repo, err := NewBackupRepo(db)
	if err != nil {
		t.Fatal(err)
	}

	id := random.UUID()
	b, err := repo.NewBackup(id)
	if err != nil {
		t.Fatal(err)
	}
	if b.AppID != id {
		t.Error("app id did not match")
	}

	backups, err := repo.GetBackups(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Error("expected 1 app")
	}
	if backups[0].AppID != id {
		t.Error("wrong id")
	}

	backup, err := repo.GetBackup(b.BackupID)
	if err != nil || backup == nil {
		t.Error("could not retrieve backup by id")
	}

	repo.CompleteBackup(b, 1234)
	backups, _ = repo.GetBackups(id)
	// PG time resolution is lower, so rounding is necessary
	if backups[0].CompletedAt.Round(time.Second) != b.CompletedAt.Round(time.Second) {
		t.Errorf("completed at time mismatch %s %s", backups[0].CompletedAt, b.CompletedAt)
	}
	if backups[0].Bytes != 1234 {
		t.Errorf("expected 1234 bytes got %d", backups[0].Bytes)
	}

	repo.DeleteBackup(b)
	backups, _ = repo.GetBackups(id)
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}
