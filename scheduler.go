package main

import (
	"log"

	"github.com/robfig/cron"
)

// 5am UTC, ~midnight EST
const defaultCronLine string = "0 0 5 * * *"

type Scheduler struct {
	PgBackups *PgBackups
	CronLine  string
	cron      *cron.Cron
}

func NewScheduler(pgBackups *PgBackups, cronLine string) *Scheduler {
	if cronLine == "" {
		cronLine = defaultCronLine
	}
	return &Scheduler{
		PgBackups: pgBackups,
		CronLine:  cronLine,
	}
}

func (s *Scheduler) Run() error {
	log.Println("Starting scheduler")

	s.cron = cron.New()
	s.cron.AddFunc(s.CronLine, s.PgBackups.BackupAll)

	s.cron.Start()

	// block forever, as cron runs in another goroutine
	select {}

	return nil
}
