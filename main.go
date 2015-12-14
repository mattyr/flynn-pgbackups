package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rlmcpherson/s3gof3r"

	"github.com/flynn/flynn/controller/client"
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/pkg/cluster"
)

type config struct {
	BucketName       string
	ControllerUrl    string
	ControllerKey    string
	ControllerTlsPin string
	Bucket           *s3gof3r.Bucket
	Client           *controller.Client
}

func main() {
	cfg := &config{}
	cfg.BucketName = os.Getenv("S3_BUCKET")
	cfg.ControllerUrl = os.Getenv("CONTROLLER_URL")
	cfg.ControllerKey = os.Getenv("CONTROLLER_KEY")
	cfg.ControllerTlsPin = os.Getenv("CONTROLLER_TLS_PIN")

	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		panic(err)
	}
	s3 := s3gof3r.New("", keys)
	cfg.Bucket = s3.Bucket(cfg.BucketName)

	if cfg.ControllerTlsPin == "" {
		c, err := controller.NewClient(cfg.ControllerUrl, cfg.ControllerKey)
		if err != nil {
			panic(err)
		}
		cfg.Client = c
	} else {
		pin, err := base64.StdEncoding.DecodeString(cfg.ControllerTlsPin)
		if err != nil {
			panic(err)
		}
		c, err := controller.NewClientWithConfig(cfg.ControllerUrl, cfg.ControllerKey, controller.Config{Pin: pin})
		if err != nil {
			panic(err)
		}
		cfg.Client = c
	}

	apps, err := cfg.Client.AppList()
	for _, a := range apps {
		r, _ := cfg.Client.GetAppRelease(a.ID)
		// TODO: log err, but continue
		if r.Env["FLYNN_POSTGRES"] != "" {
			log.Printf("Backing up %s (%s)", a.Name, a.ID)
			if err = backupApp(a, r, cfg); err != nil {
				log.Printf("Error backing up %s (%s): %s", a.Name, a.ID, err)
			} else {
				log.Printf("Completed backing up %s (%s)", a.Name, a.ID)
			}
		}
	}
}

func backupApp(app *ct.App, release *ct.Release, cfg *config) error {
	// from: https://github.com/flynn/flynn/blob/master/cli/pg.go
	pgApp := release.Env["FLYNN_POSTGRES"]
	if pgApp == "" {
		return fmt.Errorf("No postgres database found. Provision one with `flynn resource add postgres`")
	}

	pgRelease, err := cfg.Client.GetAppRelease(pgApp)
	if err != nil {
		return fmt.Errorf("error getting postgres release: %s", err)
	}

	req := &ct.NewJob{
		Entrypoint: []string{"pg_dump"},
		Cmd:        []string{"--format=custom", "--no-owner", "--no-acl"},
		TTY:        false,
		ReleaseID:  pgRelease.ID,
		ReleaseEnv: false,
		Env:        make(map[string]string),
		DisableLog: true,
	}

	for _, k := range []string{"PGHOST", "PGUSER", "PGPASSWORD", "PGDATABASE"} {
		v := release.Env[k]
		if v == "" {
			return fmt.Errorf("missing %s in app environment", k)
		}
		req.Env[k] = v
	}

	now := time.Now()
	s3Path := fmt.Sprintf("pgbackups/%s/%d.backup", app.ID, now.Unix())

	s3Putter, err := cfg.Bucket.PutWriter(s3Path, nil, nil)
	if err != nil {
		return err
	}

	rwc, err := cfg.Client.RunJobAttached(app.ID, req)
	if err != nil {
		return err
	}
	defer rwc.Close()
	defer s3Putter.Close()

	attachClient := cluster.NewAttachClient(rwc)
	attachClient.CloseWrite()

	// not worried about exit status...?
	_, err = attachClient.Receive(s3Putter, os.Stderr)

	return err
}
