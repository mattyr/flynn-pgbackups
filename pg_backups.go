package main

import (
	"log"
	"os"
	"time"

	"github.com/flynn/flynn/pkg/postgres"
)

type PgBackups struct {
	Store       Storer
	FlynnClient *FlynnClient
	Repo        *BackupRepo
}

func NewPgBackups() (*PgBackups, error) {
	db := postgres.Wait(nil, nil)
	backupRepo, err := NewBackupRepo(db)
	if err != nil {
		return nil, err
	}

	bucketName := os.Getenv("S3_BUCKET")
	store, err := NewS3Store(bucketName)
	if err != nil {
		return nil, err
	}

	c, err := NewFlynnClient()
	if err != nil {
		return nil, err
	}

	return &PgBackups{
		Repo:        backupRepo,
		FlynnClient: c,
		Store:       store,
	}, nil
}

func (pgb *PgBackups) BackupAll() {
	log.Println("Starting backups")

	apps, err := pgb.FlynnClient.AppList()
	if err != nil {
		log.Printf("Error obtaining app list: %s", err)
		return
	}

	for _, a := range apps {
		log.Printf("Backing up %s (%s)", a.App.Name, a.App.ID)
		bytes, err := pgb.BackupApp(a)
		if err != nil {
			log.Printf("Error backing up %s (%s): %s", a.App.Name, a.App.ID, err)
			pgb.DeleteOldBackups(a)
		} else {
			log.Printf("Completed backing up %s (%s) bytes: %d", a.App.Name, a.App.ID, bytes)
		}
	}
}

func (pgb *PgBackups) BackupApp(app *AppAndRelease) (int64, error) {
	var bytes int64

	// stream stdout from job to store
	var err error
	var b *Backup

	b, err = pgb.Repo.NewBackup(app.App.ID)
	if err != nil {
		return bytes, err
	}

	w, err := pgb.Store.GetPutter(app.App.ID, b.BackupID)
	if err != nil {
		return bytes, err
	}
	defer w.Close()

	err = pgb.FlynnClient.StreamBackup(app, w)
	if err != nil {
		return bytes, err
	}

	err = pgb.Repo.CompleteBackup(b, bytes)

	return bytes, err
}

func (pgb *PgBackups) DeleteOldBackups(app *AppAndRelease) error {
	backups, err := pgb.Repo.GetBackups(app.App.ID)
	if err != nil {
		return err
	}
	for _, b := range backups {
		if shouldDeleteBackup(b) {
			err = pgb.Store.Delete(b.AppID, b.BackupID)
			if err != nil {
				// just log
				log.Printf("Error deleting stored backup: %s", err)
			} else {
				err = pgb.Repo.DeleteBackup(b)
				if err != nil {
					log.Printf("Error deleting backup: %s", err)
				}
			}
		}
	}

	return nil
}

func shouldDeleteBackup(b *Backup) bool {
	// Keep those that are:
	// - less than 7 days old (one-a-day for a week)
	// - made on a sunday less than a month old (one-a-week for a month)
	// - made on the first of a month (one-a-month forever)
	// another version of this could use the backup history to determine
	// keep-ability (thereby allowing more flexible "start-of-Month" days, etc,
	// but this way the only needed inputs are an individual backup date and the
	// current date.  KISS.
	d := b.StartedAt
	now := time.Now()

	// work backwards, monthly first
	if d.Day() == 1 {
		return false
	}
	// weekly
	if (d.Weekday() == time.Sunday) && (d.After(now.Add(-32 * 24 * time.Hour))) {
		return false
	}
	// daily
	if d.After(now.Add(-8 * 24 * time.Hour)) {
		return false
	}
	return true
}
