package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/flynn/flynn/pkg/postgres"
	"github.com/robfig/cron"
)

type config struct {
	Schedule    string
	Store       Storer
	FlynnClient *FlynnClient
	Repo        *BackupRepo
}

// 5am UTC, ~midnight EST
const defaultSchedule string = "0 0 5 * * *"

func main() {
	cfg, err := createConfig()
	if err != nil {
		panic(err)
	}

	c := cron.New()

	c.AddFunc(cfg.Schedule, func() { doBackups(cfg) })
	log.Println("Starting cron")
	c.Start()
	// block forever, as cron runs in another goroutine
	select {}
}

func createConfig() (*config, error) {
	db := postgres.Wait(nil, nil)
	backupRepo, err := NewBackupRepo(db)
	if err != nil {
		return nil, err
	}

	sched := os.Getenv("SCHEDULE")
	if sched == "" {
		sched = defaultSchedule
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

	return &config{
		Repo:        backupRepo,
		Schedule:    sched,
		FlynnClient: c,
		Store:       store,
	}, nil
}

func doBackups(cfg *config) {
	log.Println("Starting backups")

	apps, err := cfg.FlynnClient.AppList()
	if err != nil {
		log.Printf("Error obtaining app list: %s", err)
		return
	}

	for _, a := range apps {
		log.Printf("Backing up %s (%s)", a.App.Name, a.App.ID)
		bytes, err := backupApp(a, cfg)
		if err != nil {
			log.Printf("Error backing up %s (%s): %s", a.App.Name, a.App.ID, err)
			deleteOldBackups(a, cfg)
		} else {
			log.Printf("Completed backing up %s (%s) bytes: %d", a.App.Name, a.App.ID, bytes)
		}
	}
}

func backupApp(app *AppAndRelease, cfg *config) (int64, error) {
	var bytes int64

	// stream stdout from job to store
	var err error
	var b *Backup

	b, err = cfg.Repo.NewBackup(app.App.ID)
	if err != nil {
		return bytes, err
	}

	r, w := io.Pipe()

	go func() {
		bytes, err = cfg.Store.Put(app.App.ID, b.BackupID, r)
	}()

	err = cfg.FlynnClient.StreamBackup(app, w)
	if err != nil {
		return bytes, err
	}

	err = cfg.Repo.CompleteBackup(b, bytes)

	return bytes, err
}

func deleteOldBackups(app *AppAndRelease, cfg *config) error {
	backups, err := cfg.Repo.GetBackups(app.App.ID)
	if err != nil {
		return err
	}
	for _, b := range backups {
		if shouldDeleteBackup(b) {
			err = cfg.Store.Delete(b.AppID, b.BackupID)
			if err != nil {
				// just log
				log.Printf("Error deleting stored backup: %s", err)
			} else {
				err = cfg.Repo.DeleteBackup(b)
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
