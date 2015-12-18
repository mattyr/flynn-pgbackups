package main

import (
	"github.com/flynn/flynn/pkg/postgres"
)

func migrate(db *postgres.DB) error {
	m := postgres.NewMigrations()

	m.Add(1,
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		`CREATE TABLE pgbackups (
		app_id uuid NOT NULL,
		backup_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
		started_at timestamptz NOT NULL DEFAULT now(),
		completed_at timestamptz,
		bytes bigint NOT NULL DEFAULT 0
	)`,

		`CREATE INDEX ON pgbackups (app_id)`)

	return m.Migrate(db)
}
