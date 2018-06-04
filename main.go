package main

import (
	"fmt"
	"os"
)

func main() {
	pgb, err := NewPgBackups()
	if err != nil {
		panic(err)
	}

	action := os.Args[1]

	switch action {
	case "worker":
		runScheduler(pgb)
		break
	case "run":
		pgb.BackupAll()
		break
	case "list":
		listBackups(pgb)
		break
	case "url":
		backupUrl(pgb)
		break
	}
	os.Exit(0)
}

func runScheduler(pgb *PgBackups) {
	s := NewScheduler(pgb, os.Getenv("SCHEDULE"))
	err := s.Run()
	if err != nil {
		panic(err)
	}
}

func listBackups(pgb *PgBackups) {
	if len(os.Args) < 3 {
		panic("App name must be given (pgbackups [list] [appname])")
	}

	appName := os.Args[2]

	if appName == "" {
		panic("App name must be given (pgbackups list [appname])")
	}

	app, err := pgb.FlynnClient.GetApp(appName)

	if err != nil {
		panic(err)
	}

	backups, err := pgb.Repo.GetBackups(app.ID)

	if err != nil {
		panic(err)
	}

	fmt.Printf("App: %s ID: %s\n", app.Name, app.ID)
	fmt.Println("  [ID] - [Started] - [Completed] - [Bytes]")
	for _, b := range backups {
		fmt.Printf("  %s - %s - %s - %d\n", b.BackupID, b.StartedAt, b.CompletedAt, b.Bytes)
	}
}

func backupUrl(pgb *PgBackups) {
	if len(os.Args) < 3 {
		panic("Backup id must be given (pgbackups [url] [backup id])")
	}
	id := os.Args[2]
	if id == "" {
		panic("Backup id must be given (pgbackups [url] [backup id])")
	}

	b, err := pgb.Repo.GetBackup(id)
	if err != nil || b == nil {
		panic(err)
	}

	url, err := pgb.Store.DownloadUrl(b.AppID, b.BackupID)
	if err != nil {
		panic(err)
	}
	fmt.Println(url)
}
