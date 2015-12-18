package main

import (
	"time"

	"github.com/flynn/flynn/pkg/postgres"
	"github.com/flynn/flynn/pkg/random"
)

type Backup struct {
	AppID       string
	BackupID    string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Bytes       int64
}

type BackupRepo struct {
	db *postgres.DB
}

func NewBackupRepo(db *postgres.DB) (*BackupRepo, error) {
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &BackupRepo{db: db}, nil
}

func (r *BackupRepo) NewBackup(appID string) (*Backup, error) {
	now := time.Now()
	b := &Backup{
		AppID:       appID,
		BackupID:    random.UUID(),
		StartedAt:   &now,
		CompletedAt: nil,
		Bytes:       0,
	}

	err := r.db.Exec("INSERT INTO pgbackups (app_id, backup_id, started_at, completed_at, bytes) VALUES ($1, $2, $3, $4, $5)",
		b.AppID, b.BackupID, b.StartedAt, nil, 0)

	return b, err
}

func (r *BackupRepo) GetBackups(appID string) ([]*Backup, error) {
	rows, err := r.db.Query("SELECT * FROM pgbackups WHERE app_id = $1", appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	backups := []*Backup{}
	for rows.Next() {
		b := &Backup{}
		err := rows.Scan(&b.AppID, &b.BackupID, &b.StartedAt, &b.CompletedAt, &b.Bytes)
		if err != nil {
			rows.Close()
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

func (r *BackupRepo) CompleteBackup(b *Backup, bytes int64) error {
	now := time.Now()
	b.CompletedAt = &now
	b.Bytes = bytes
	err := r.db.Exec("UPDATE pgbackups SET completed_at = $1, bytes = $2 WHERE backup_id = $3", now, b.Bytes, b.BackupID)

	return err
}

func (r *BackupRepo) DeleteBackup(b *Backup) error {
	return r.db.Exec("DELETE FROM pgbackups WHERE backup_id = $1", b.BackupID)
}
